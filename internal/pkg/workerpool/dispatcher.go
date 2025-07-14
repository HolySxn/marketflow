package workerpool

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	// If workes idle for at least this period of time, then stop a worker.
	idleTimeout = 2 * time.Second
)

type WorkerPool struct {
	maxWorkers  int
	taskQueue   chan func()
	workerQueue chan func()
	stoppedChan chan struct{}
	stopOnce    sync.Once
	metrics     struct {
		workersActive atomic.Int32
		tasksDropped  atomic.Int32
	}
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	pool := &WorkerPool{
		maxWorkers:  maxWorkers,
		taskQueue:   make(chan func(), 1000),
		workerQueue: make(chan func()),
		stoppedChan: make(chan struct{}),
	}

	go pool.dispatch()

	return pool
}

// Submit enqueues a function for a worker to execute.
func (p *WorkerPool) Submit(task func()) {
	if task != nil {
		select {
		case p.taskQueue <- task:
		default:
			p.metrics.tasksDropped.Add(1)
		}
	}
}

// Stop stops the workerpool and waits
func (p *WorkerPool) Stop() {
	p.stop()
}

func worker(task func(), workerQueue chan func(), wg *sync.WaitGroup, workersActive *atomic.Int32) {
	defer func() {
		workersActive.Add(-1)
		wg.Done()
	}()
	for task != nil {
		task()
		task = <-workerQueue
	}
}

func (p *WorkerPool) dispatch() {
	defer close(p.stoppedChan)
	var wg sync.WaitGroup
	timeout := time.NewTimer(idleTimeout)
	defer timeout.Stop()

	var workerCount int
	var idle bool

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				// Stop workers
				for workerCount > 0 {
					p.workerQueue <- nil
					workerCount--
				}
				wg.Wait()

				return
			}

			select {
			case p.workerQueue <- task:
			default:
				if workerCount < p.maxWorkers {
					wg.Add(1)
					go worker(task, p.workerQueue, &wg, &p.metrics.workersActive)
					workerCount++
					p.metrics.workersActive.Add(1)
				} else {
					p.metrics.tasksDropped.Add(1)
				}
				idle = false
			}
		case <-timeout.C:
			if idle && workerCount > 0 {
				p.workerQueue <- nil
				workerCount--
			}
			idle = true
			timeout.Reset(idleTimeout)
		}
	}
}

func (p *WorkerPool) stop() {
	p.stopOnce.Do(func() {
		close(p.taskQueue)
	})
	<-p.stoppedChan
}

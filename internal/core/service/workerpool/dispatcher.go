package workerpool

// import (
// 	"sync"
// )

// type WorkerPool struct {
// 	maxWorkers  int
// 	tasks       chan func()
// 	workerQueue chan func()
// 	stoppedChan chan struct{}
// }

// func worker(workerQueue chan func(), wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	for task := range workerQueue {
// 		task()
// 	}
// }

// func (p *WorkerPool) dispatch() {
// 	defer close(p.stoppedChan)
// 	wg := &sync.WaitGroup{}

// 	for i := 0; i < p.maxWorkers; i++ {
// 		wg.Add(1)
// 		go worker(p.workerQueue, wg)
// 	}

// 	for {
// 		select{
// 		case task, ok := <- p.tasks:
// 			if !ok {
// 				break
// 			}

// 		}
// 	}
// }

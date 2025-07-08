// package main

// import (
// 	"context"
// 	"log/slog"
// 	"marketflow/config"
// 	"net"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"sync"
// 	"time"
// )

// func init() {
// 	initialLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
// 	slog.SetDefault(initialLogger)
// }

// func main() {
// 	ctx := context.Background()
// 	cfg := config.LoadConfig()

// 	deps, err := config.NewDependencies(
// 		ctx,
// 		config.WithLogger(cfg.Server.LogLvl),
// 		config.WithPostgres(
// 			cfg.Postgres.User,
// 			cfg.Postgres.Pass,
// 			cfg.Postgres.Host,
// 			cfg.Postgres.Port,
// 			cfg.Postgres.DBName,
// 		),
// 		config.WithRedis(
// 			cfg.Redis.Addr,
// 			cfg.Redis.DB,
// 		),
// 	)

// 	if err != nil {
// 		slog.Error("failed to load dependencies", slog.Any("error", err))
// 		os.Exit(1)
// 	}
// 	defer deps.Close()
// 	slog.SetDefault(deps.Logger)
// }

// func run(ctx context.Context, cfg *config.Config, srv http.Handler) {
// 	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
// 	defer cancel()

// 	httpServer := &http.Server{
// 		Addr:    net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
// 		Handler: srv,
// 	}

// 	go func() {
// 		slog.Info("server listening", "address", httpServer.Addr)
// 		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 			slog.Info("error listening and serving", "error", err)
// 		}
// 	}()

// 	wg := sync.WaitGroup{}
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		<-ctx.Done()
// 		shutdownCtx := context.Background()
// 		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
// 		defer cancel()
// 		slog.Info("Gracefully shutting down...")

// 		if err := httpServer.Shutdown(shutdownCtx); err != nil {
// 			slog.Info("error shutting down http server", "error", err)
// 		}
// 	}()
// 	wg.Wait()
// }

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:40101")
	if err != nil {
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			break
		}

		fmt.Println(string(line))
	}
}

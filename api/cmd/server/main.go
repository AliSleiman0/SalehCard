package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/AliSleiman0/salehcard/api/internal/config"
	mongoplatform "github.com/AliSleiman0/salehcard/api/internal/platform/mongo"
	"github.com/AliSleiman0/salehcard/api/internal/server"
)

func main() {
	// Load .env file if present; ignore error if file does not exist.
	_ = godotenv.Load()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "env", cfg.Env, "error", err)
		os.Exit(1)
	}

	slog.Info("connecting to MongoDB", "uri", cfg.MongoURI, "db", cfg.DBName)

	client, db, err := mongoplatform.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		slog.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer mongoplatform.Disconnect(context.Background(), client)

	slog.Info("connected to MongoDB")

	srv := server.New(cfg, db)
	srv.Routes()

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the HTTP server in a goroutine.
	go func() {
		slog.Info("starting HTTP server", "port", cfg.Port, "env", cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Background workers (currently just the USDT payment watcher) run under a
	// cancellable context and are awaited on shutdown. An interrupted tick is
	// safe by design: claimed-but-unsettled intents stay `confirming` and are
	// retried on the next boot's first tick.
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	if w := srv.Watcher(); w != nil {
		workers.Add(1)
		go func() {
			defer workers.Done()
			w.Run(workerCtx)
		}()
	}

	// Wait for interrupt signal to gracefully shut down the server.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	stopWorkers()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced server shutdown", "error", err)
		os.Exit(1)
	}
	workers.Wait()

	slog.Info("server exited cleanly")
}

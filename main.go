package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iamdanhart/te-live/config"
	"github.com/iamdanhart/te-live/queue"
	"github.com/iamdanhart/te-live/router"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q, err := queue.NewPgQueue(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router.NewRouter(ctx, cfg, q),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		slog.Info("Listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	if err := q.Close(); err != nil {
		slog.Error("db close error", "err", err)
	}
}

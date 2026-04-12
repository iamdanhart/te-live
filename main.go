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
	"github.com/iamdanhart/te-live/router"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	srv := &http.Server{Addr: ":8080", Handler: router.NewRouter(cfg)}

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

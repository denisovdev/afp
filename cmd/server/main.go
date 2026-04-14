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

	"github.com/dxngee/antifraud-processing/internal/config"
	handler "github.com/dxngee/antifraud-processing/internal/http"
	"github.com/dxngee/antifraud-processing/internal/service"
	"github.com/dxngee/antifraud-processing/internal/storage/postgres"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrationsDir := envOrDefault("MIGRATIONS_DIR", "migrations")
	if err := postgres.RunMigrations(ctx, pool, migrationsDir); err != nil {
		return err
	}

	profileRepo := postgres.NewProfileRepo(pool)
	eventRepo := postgres.NewEventRepo(pool)
	linkRepo := postgres.NewLinkRepo(pool)
	clusterRepo := postgres.NewClusterRepo(pool)
	txm := postgres.NewTxManager(pool)

	svc := service.NewFingerprintService(profileRepo, eventRepo, linkRepo, clusterRepo, txm, cfg.MatchThreshold)
	h := handler.NewHandler(svc)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      h.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

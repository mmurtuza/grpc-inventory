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

	db "github.com/murtuza/grpc-inventory/db/sqlc"
	"github.com/murtuza/grpc-inventory/internal/cache"
	"github.com/murtuza/grpc-inventory/internal/config"
	"github.com/murtuza/grpc-inventory/internal/inventory"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(newLogger(cfg.LogFormat))

	ctx := context.Background()

	// ── PostgreSQL ─────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	// ── Redis ──────────────────────────────────────────────────────────────────
	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to init redis client", "err", err)
		os.Exit(1)
	}
	var c cache.Cache = redisCache
	if err := redisCache.Ping(ctx); err != nil {
		slog.Warn("redis unavailable, falling back to no-op cache", "err", err)
		c = cache.NoopCache{}
	} else {
		slog.Info("connected to Redis")
	}

	// ── Handler & router ───────────────────────────────────────────────────────
	h := inventory.NewHandler(inventory.NewPgStore(db.New(pool)), c, pool)
	router := inventory.NewRouter(h, cfg.AllowedOrigin)

	// ── HTTP server ────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.RestAPIPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("REST Inventory API listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// ── Graceful shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down REST API server...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	slog.Info("REST API server stopped")
}

func newLogger(format string) *slog.Logger {
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

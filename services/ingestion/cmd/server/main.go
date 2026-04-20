package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/activeprospect/multi-tenant-billing/ingestion/internal/handler"
	"github.com/activeprospect/multi-tenant-billing/ingestion/internal/kafka"
	rdb "github.com/activeprospect/multi-tenant-billing/ingestion/internal/redis"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	producer, err := kafka.NewProducer(strings.Split(mustEnv("KAFKA_BROKERS"), ","))
	if err != nil {
		logger.Error("failed to create kafka producer", "err", err)
		os.Exit(1)
	}
	defer producer.Close()

	idempotency := rdb.NewIdempotencyStore(mustEnv("REDIS_URL"))
	eventHandler := handler.NewEventHandler(idempotency, producer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/events", eventHandler.IngestEvent)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("ingestion service listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

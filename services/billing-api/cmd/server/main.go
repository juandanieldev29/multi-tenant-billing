package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/db"
	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/handler"
	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/pricing"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbClient, err := db.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	engine := pricing.New()
	billingHandler := handler.NewBillingHandler(dbClient, engine, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/tenants/{id}/invoice", billingHandler.GetInvoice)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("billing-api listening", "addr", srv.Addr)
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

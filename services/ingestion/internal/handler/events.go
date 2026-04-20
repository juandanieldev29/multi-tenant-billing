package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/activeprospect/multi-tenant-billing/ingestion/internal/kafka"
	"github.com/activeprospect/multi-tenant-billing/ingestion/internal/model"
	rdb "github.com/activeprospect/multi-tenant-billing/ingestion/internal/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("ingestion/handler")

type EventHandler struct {
	idempotency *rdb.IdempotencyStore
	producer    *kafka.Producer
	logger      *slog.Logger
}

func NewEventHandler(idempotency *rdb.IdempotencyStore, producer *kafka.Producer, logger *slog.Logger) *EventHandler {
	return &EventHandler{idempotency: idempotency, producer: producer, logger: logger}
}

func (h *EventHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "IngestEvent")
	defer span.End()

	var event model.BillingEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if event.EventID == "" || event.TenantID == "" || event.Type == "" {
		http.Error(w, "event_id, tenant_id, and type are required", http.StatusBadRequest)
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	span.SetAttributes(
		attribute.String("tenant.id", event.TenantID),
		attribute.String("event.id", event.EventID),
		attribute.String("event.type", event.Type),
	)

	acquired, err := h.idempotency.Acquire(ctx, event.TenantID, event.EventID)
	if err != nil {
		h.logger.ErrorContext(ctx, "idempotency check failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !acquired {
		w.WriteHeader(http.StatusOK) // duplicate — already processed
		return
	}

	if err := h.producer.Publish(ctx, &event); err != nil {
		h.logger.ErrorContext(ctx, "kafka publish failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

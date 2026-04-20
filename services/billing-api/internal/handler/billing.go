package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/db"
	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/model"
	"github.com/activeprospect/multi-tenant-billing/billing-api/internal/pricing"
)

type BillingHandler struct {
	db     *db.Client
	engine *pricing.Engine
	logger *slog.Logger
}

func NewBillingHandler(db *db.Client, engine *pricing.Engine, logger *slog.Logger) *BillingHandler {
	return &BillingHandler{db: db, engine: engine, logger: logger}
}

func (h *BillingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	if tenantID == "" {
		http.Error(w, "tenant id required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	aggs, err := h.db.GetUsageAggregates(r.Context(), tenantID, periodStart, now)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "db query failed", "err", err, "tenant_id", tenantID)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	invoice := &model.Invoice{
		TenantID:    tenantID,
		PeriodStart: periodStart,
		PeriodEnd:   now,
		GeneratedAt: time.Now().UTC(),
	}

	for _, agg := range aggs {
		total, err := h.engine.Calculate(agg.PricingConfig, agg.TotalQty, agg.Seats)
		if err != nil {
			h.logger.WarnContext(r.Context(), "pricing calculation failed", "err", err)
			continue
		}
		invoice.LineItems = append(invoice.LineItems, model.LineItem{
			Description: agg.EventType,
			Quantity:    agg.TotalQty,
			UnitPrice:   total / max(1, int64(agg.TotalQty)),
			Total:       total,
		})
		invoice.TotalCents += total
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoice)
}

package model

import "time"

type LineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   int64   `json:"unit_price_cents"`
	Total       int64   `json:"total_cents"`
}

type Invoice struct {
	TenantID    string     `json:"tenant_id"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	LineItems   []LineItem `json:"line_items"`
	TotalCents  int64      `json:"total_cents"`
	GeneratedAt time.Time  `json:"generated_at"`
}

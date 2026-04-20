package model

import "time"

type BillingEvent struct {
	EventID    string            `json:"event_id"`
	TenantID   string            `json:"tenant_id"`
	Type       string            `json:"type"`
	Quantity   float64           `json:"quantity"`
	ResourceID string            `json:"resource_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

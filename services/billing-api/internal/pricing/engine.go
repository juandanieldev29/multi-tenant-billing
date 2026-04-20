package pricing

import (
	"encoding/json"
	"fmt"
)

type PricingType string

const (
	PerRequest PricingType = "per_request"
	Tiered     PricingType = "tiered"
	SeatBased  PricingType = "seat_based"
)

type PricingConfig struct {
	Type           PricingType `json:"type"`
	UnitPriceCents int64       `json:"unit_price_cents,omitempty"`
	Tiers          []Tier      `json:"tiers,omitempty"`
	SeatPriceCents int64       `json:"seat_price_cents,omitempty"`
}

type Tier struct {
	UpTo           *float64 `json:"up_to"` // nil = unlimited
	UnitPriceCents int64    `json:"unit_price_cents"`
}

type Engine struct{}

func New() *Engine { return &Engine{} }

func (e *Engine) Calculate(configJSON []byte, quantity float64, seats int) (int64, error) {
	var cfg PricingConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return 0, fmt.Errorf("parse pricing config: %w", err)
	}

	switch cfg.Type {
	case PerRequest:
		return int64(quantity) * cfg.UnitPriceCents, nil
	case Tiered:
		return calculateTiered(cfg.Tiers, quantity), nil
	case SeatBased:
		return int64(seats) * cfg.SeatPriceCents, nil
	default:
		return 0, fmt.Errorf("unknown pricing type: %s", cfg.Type)
	}
}

func calculateTiered(tiers []Tier, quantity float64) int64 {
	var total int64
	remaining := quantity
	for _, tier := range tiers {
		if remaining <= 0 {
			break
		}
		var tierQty float64
		if tier.UpTo == nil {
			tierQty = remaining
		} else {
			tierQty = min(remaining, *tier.UpTo)
		}
		total += int64(tierQty) * tier.UnitPriceCents
		remaining -= tierQty
	}
	return total
}

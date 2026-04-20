package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageAggregate struct {
	TenantID      string
	EventType     string
	WindowStart   time.Time
	WindowEnd     time.Time
	TotalQty      float64
	PricingConfig []byte
	Seats         int
}

type Client struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Client, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	return &Client{pool: pool}, nil
}

// GetUsageAggregates returns daily aggregates for a tenant within [from, to].
// tenant_id is the leading predicate so Citus routes the query to a single shard group.
func (c *Client) GetUsageAggregates(ctx context.Context, tenantID string, from, to time.Time) ([]UsageAggregate, error) {
	rows, err := c.pool.Query(ctx, `
		SELECT ua.tenant_id, ua.event_type, ua.window_start, ua.window_end,
		       ua.total_quantity, pp.config, t.seats
		FROM   usage_aggregates ua
		JOIN   tenants       t  ON  t.id          = ua.tenant_id
		JOIN   pricing_plans pp ON  pp.tenant_id  = ua.tenant_id
		                        AND pp.event_type  = ua.event_type
		WHERE  ua.tenant_id      = $1
		  AND  ua.window_start  >= $2
		  AND  ua.window_end    <= $3
		  AND  ua.window_duration = '1 day'
		ORDER  BY ua.window_start
	`, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query aggregates: %w", err)
	}
	defer rows.Close()

	var aggs []UsageAggregate
	for rows.Next() {
		var a UsageAggregate
		if err := rows.Scan(&a.TenantID, &a.EventType, &a.WindowStart, &a.WindowEnd,
			&a.TotalQty, &a.PricingConfig, &a.Seats); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, rows.Err()
}

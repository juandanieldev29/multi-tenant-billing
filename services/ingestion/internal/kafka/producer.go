package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/IBM/sarama"
	"github.com/activeprospect/multi-tenant-billing/ingestion/internal/model"
	"go.opentelemetry.io/otel/propagation"
)

const topic = "billing.events"

type Producer struct {
	producer sarama.SyncProducer
}

func NewProducer(brokers []string) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Return.Successes = true
	cfg.Producer.Idempotent = true
	cfg.Net.MaxOpenRequests = 1

	hostname, _ := os.Hostname()
	cfg.Producer.Transaction.ID = fmt.Sprintf("ingestion-%s", hostname)

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}
	return &Producer{producer: p}, nil
}

func (p *Producer) Publish(ctx context.Context, event *model.BillingEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	carrier := propagation.MapCarrier{}
	propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}).Inject(ctx, carrier)

	headers := make([]sarama.RecordHeader, 0, len(carrier))
	for k, v := range carrier {
		headers = append(headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.StringEncoder(event.TenantID),
		Value:   sarama.ByteEncoder(payload),
		Headers: headers,
	}

	if err := p.producer.BeginTxn(); err != nil {
		return fmt.Errorf("begin txn: %w", err)
	}
	if _, _, err = p.producer.SendMessage(msg); err != nil {
		_ = p.producer.AbortTxn()
		return fmt.Errorf("send message: %w", err)
	}
	return p.producer.CommitTxn()
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

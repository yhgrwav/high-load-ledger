package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"high-load-ledger/internal/domain/entity"
)

type TransactionPublisher interface {
	PublishTransaction(ctx context.Context, tx entity.Transaction) error
}

type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewProducer(writer *kafka.Writer, logger *slog.Logger) *Producer {
	return &Producer{writer: writer, logger: logger}
}

func (p *Producer) PublishTransaction(ctx context.Context, tx entity.Transaction) error {
	payload, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("kafka: marshal transaction: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   tx.ID[:],
		Value: payload,
	})
	if err != nil {
		p.logger.ErrorContext(ctx, "kafka: publish transaction failed",
			"err", err,
			"transaction_id", tx.ID,
		)
		return fmt.Errorf("kafka: publish transaction: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

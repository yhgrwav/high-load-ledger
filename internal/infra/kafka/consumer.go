package kafka

import (
	"context"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, msg kafka.Message) error

type Consumer struct {
	reader *kafka.Reader
	logger *slog.Logger
}

func NewConsumer(reader *kafka.Reader, logger *slog.Logger) *Consumer {
	return &Consumer{reader: reader, logger: logger}
}

func (c *Consumer) Run(ctx context.Context, handle Handler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		if err := handle(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "kafka: handle message failed",
				"err", err,
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "kafka: commit failed", "err", err)
			return err
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

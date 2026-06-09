package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
)

type TransactionCacheProjector struct {
	cache  repository.CacheRepository
	ttl    time.Duration
	logger *slog.Logger
}

func NewTransactionCacheProjector(cache repository.CacheRepository, ttl time.Duration, logger *slog.Logger) *TransactionCacheProjector {
	return &TransactionCacheProjector{
		cache:  cache,
		ttl:    ttl,
		logger: logger,
	}
}

func (p *TransactionCacheProjector) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	var tx entity.Transaction
	if err := json.Unmarshal(msg.Value, &tx); err != nil {
		return fmt.Errorf("projector: decode transaction: %w", err)
	}
	if tx.ID == uuid.Nil && len(msg.Key) == 16 {
		id, err := uuid.FromBytes(msg.Key)
		if err != nil {
			return fmt.Errorf("projector: invalid message key: %w", err)
		}
		tx.ID = id
	}
	if tx.ID == uuid.Nil {
		return fmt.Errorf("projector: empty transaction id")
	}

	if err := p.cache.SetTransaction(ctx, &tx, p.ttl); err != nil {
		return err
	}

	p.logger.InfoContext(ctx, "projector: transaction cached", "transaction_id", tx.ID, "partition", msg.Partition, "offset", msg.Offset)
	return nil
}

package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type PostingRepository interface {
	CreateBatch(ctx context.Context, postings []entity.Posting) error
	ListByAccountID(ctx context.Context, accountID uuid.UUID) ([]entity.Posting, error)
}

package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type PostingRepository interface {
	CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error
	ListPostingsByAccountID(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]entity.Posting, error)
	GetBalanceFromPostings(ctx context.Context, accountID uuid.UUID) (int64, error)
}

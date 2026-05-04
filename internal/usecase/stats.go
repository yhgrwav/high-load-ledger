package usecase

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
)

type StatsUseCase struct {
	repo   repository.TransactionRepository
	logger *slog.Logger
}

func NewStatsUseCase(repo repository.TransactionRepository, logger *slog.Logger) *StatsUseCase {
	return &StatsUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (s *StatsUseCase) GetTransaction(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	return s.repo.GetTransactionByID(ctx, id)
}

func (s *StatsUseCase) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, tx entity.CustomTx, status entity.TransactionStatus) error {
	return s.repo.UpdateStatus(ctx, tx, id, status)
}

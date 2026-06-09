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
	cache  repository.CacheRepository
	logger *slog.Logger
}

func NewStatsUseCase(
	repo repository.TransactionRepository,
	cache repository.CacheRepository,
	logger *slog.Logger,
) *StatsUseCase {
	return &StatsUseCase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (s *StatsUseCase) GetTransaction(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	tx, err := s.cache.GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		return tx, nil
	}
	return s.repo.GetTransactionByID(ctx, id)
}

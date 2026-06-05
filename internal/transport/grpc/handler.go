package grpc

import (
	"log/slog"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/usecase"
)

type Handler struct {
	ledger.UnimplementedTransactionServiceServer
	ledger.UnimplementedAccountServiceServer
	ledger.UnimplementedStatsServiceServer
	transferUC *usecase.TransferUseCase
	accountUC  *usecase.AccountUseCase
	statsUC    *usecase.StatsUseCase
	logger     *slog.Logger
}

func NewHandler(
	transferUC *usecase.TransferUseCase,
	accountUC *usecase.AccountUseCase,
	statsUC *usecase.StatsUseCase,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		transferUC: transferUC,
		accountUC:  accountUC,
		statsUC:    statsUC,
		logger:     logger,
	}
}

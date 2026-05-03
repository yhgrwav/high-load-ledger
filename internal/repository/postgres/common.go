package postgres

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/jackc/pgx/v5"
)

// castTx - локальная вспомогательный метод, который будет типизировать тип any в pgx.Tx (конкретно в этой реализации)
// и будет возвращать сущность, с которой будет удобно работать внутри других методов, а также этот метод поможет избежать
// дублирования кода, что способствует реализации принципа DRY
func (db *Repository) castTx(ctx context.Context, tx entity.CustomTx) (pgx.Tx, error) {
	if tx == nil {
		db.logger.ErrorContext(ctx, "db: failed to cast tx")
		return nil, entity.ErrInvalidTxType
	}
	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, entity.ErrInvalidTxType
	}
	return pgTx, nil
}

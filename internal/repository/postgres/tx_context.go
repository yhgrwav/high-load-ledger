package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txContextKey struct{}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func querierFromContext(ctx context.Context, pool *Repository) querier {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool.pool
}

func (db *Repository) ReadWrite(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to begin read-write transaction", "err", err)
		return fmt.Errorf("db: failed to begin read-write transaction: %w", err)
	}

	txCtx := withTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			db.logger.ErrorContext(ctx, "db: rollback failed", "err", rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		db.logger.ErrorContext(ctx, "db: commit failed", "err", err)
		return fmt.Errorf("db: commit failed: %w", err)
	}
	return nil
}

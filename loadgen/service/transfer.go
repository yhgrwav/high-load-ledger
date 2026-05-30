package service

import (
	"context"
	"fmt"

	gen "high-load-ledger/gen/go"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type TxManager struct {
	conn *grpc.ClientConn
}

func NewTxManager(conn *grpc.ClientConn) *TxManager {
	return &TxManager{conn: conn}
}

func (t *TxManager) CreateTx(ctx context.Context, currency gen.Currency, userFrom, userTo []byte, amount int64) (uuid.UUID, error) {
	ik, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate idempotency key: %w", err)
	}

	req := &gen.TransferRequest{
		IdempotencyKey: ik[:],
		UserFromId:     userFrom,
		UserToId:       userTo,
		Amount:         amount,
		Currency:       currency,
	}

	txResult, err := gen.NewTransactionServiceClient(t.conn).Transfer(ctx, req)
	if err != nil {
		return uuid.Nil, err
	}

	txID, err := uuid.FromBytes(txResult.TransactionId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse transaction id: %w", err)
	}

	return txID, nil
}

package service

import (
	"context"
	"fmt"
	"sync/atomic"

	gen "high-load-ledger/gen/go"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type TxManager struct {
	conns []*grpc.ClientConn
	rr    atomic.Uint64
}

func NewTxManager(conns ...*grpc.ClientConn) *TxManager {
	return &TxManager{conns: conns}
}

func (t *TxManager) conn() *grpc.ClientConn {
	if len(t.conns) == 1 {
		return t.conns[0]
	}
	i := t.rr.Add(1) - 1
	return t.conns[int(i%uint64(len(t.conns)))]
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

	txResult, err := gen.NewTransactionServiceClient(t.conn()).Transfer(ctx, req)
	if err != nil {
		return uuid.Nil, err
	}

	txID, err := uuid.FromBytes(txResult.TransactionId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse transaction id: %w", err)
	}

	return txID, nil
}

func (t *TxManager) GetTransaction(ctx context.Context, txID uuid.UUID) error {
	req := &gen.GetTransactionRequest{
		TransactionId: txID[:],
	}
	_, err := gen.NewStatsServiceClient(t.conn()).GetTransaction(ctx, req)
	return err
}

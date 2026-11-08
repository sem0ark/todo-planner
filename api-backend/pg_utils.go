package main

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgBatchSender interface {
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

type pgBatchBuilder struct {
	batch pgx.Batch
}

func newPgBatchBuilder() *pgBatchBuilder {
	return &pgBatchBuilder{}
}

func (builder *pgBatchBuilder) Queue(query string, arguments ...any) *pgBatchBuilder {
	builder.batch.Queue(query, arguments...)
	return builder
}

func (builder *pgBatchBuilder) Execute(
	ctx context.Context,
	sender pgBatchSender,
	consumeResults func(pgx.BatchResults) error,
) error {
	batchResults := sender.SendBatch(ctx, &builder.batch)
	consumeError := consumeResults(batchResults)
	closeError := batchResults.Close()
	if consumeError != nil {
		return consumeError
	}
	return closeError
}

// dbQuerier is satisfied by both *pgxpool.Pool and pgx.Tx, so read queries
// need no separate transactional copy.
type dbQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

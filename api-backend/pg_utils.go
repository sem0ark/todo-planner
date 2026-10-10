package main

import (
	"context"

	"github.com/jackc/pgx/v5"
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

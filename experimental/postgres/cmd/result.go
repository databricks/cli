package postgrescmd

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// unitResult is the buffered result of running one input unit. The
// multi-input renderers (text, json) need rows buffered before they can
// emit a per-unit block; for the single-input path we still stream
// directly through a rowSink and never produce a unitResult.
type unitResult struct {
	Source     string
	SQL        string
	Fields     []pgconn.FieldDescription
	Rows       [][]any
	CommandTag string
	Elapsed    time.Duration
}

// IsRowsProducing returns whether the unit returned a row description.
func (r *unitResult) IsRowsProducing() bool {
	return len(r.Fields) > 0
}

// runUnitBuffered runs sql and collects every row into memory. Implemented
// as a thin wrapper that hands a bufferSink to executeOne, so error wrapping
// and the rowSink contract stay in one place rather than parallel-evolving
// across two query loops.
func runUnitBuffered(ctx context.Context, conn *pgx.Conn, unit inputUnit) (*unitResult, error) {
	start := time.Now()
	r := &unitResult{Source: unit.Source, SQL: unit.SQL}
	sink := &bufferSink{result: r}
	if err := executeOne(ctx, conn, unit.SQL, sink); err != nil {
		// Capture timing on the error path too so the JSON error envelope
		// can surface "this query ran for X seconds before failing".
		r.Elapsed = time.Since(start)
		return r, err
	}
	r.Elapsed = time.Since(start)
	return r, nil
}

// bufferSink is a rowSink that copies fields, rows, and the command tag into
// a unitResult instead of writing anywhere. Used by the multi-input path so
// successive units can be rendered together once they're all available.
type bufferSink struct {
	result *unitResult
}

func (s *bufferSink) Begin(fields []pgconn.FieldDescription) error {
	s.result.Fields = fields
	return nil
}

func (s *bufferSink) Row(values []any) error {
	s.result.Rows = append(s.result.Rows, values)
	return nil
}

func (s *bufferSink) End(commandTag string) error {
	s.result.CommandTag = commandTag
	return nil
}

func (s *bufferSink) OnError(err error) {}

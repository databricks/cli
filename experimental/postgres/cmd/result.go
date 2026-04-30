package postgrescmd

import (
	"context"
	"fmt"
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

// runUnitBuffered runs sql and collects every row into memory. Used by the
// multi-input output paths (text and json), where per-unit buffering is
// acceptable per the plan: a multi-input invocation that emits a huge
// SELECT will buffer that result before printing. Users with huge result
// sets per statement should use single-input invocations (which fully
// stream) or --output csv on a single input.
func runUnitBuffered(ctx context.Context, conn *pgx.Conn, unit inputUnit) (*unitResult, error) {
	start := time.Now()
	rows, err := conn.Query(ctx, unit.SQL, pgx.QueryExecModeExec)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	r := &unitResult{
		Source: unit.Source,
		SQL:    unit.SQL,
		Fields: rows.FieldDescriptions(),
	}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("decode row: %w", err)
		}
		r.Rows = append(r.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	r.CommandTag = rows.CommandTag().String()
	r.Elapsed = time.Since(start)
	return r, nil
}

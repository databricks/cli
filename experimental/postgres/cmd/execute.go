package postgrescmd

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// rowSink consumes a query result one row at a time. Sinks that maintain open
// output structures (e.g. a streaming JSON array) implement OnError so they
// can close cleanly when the iteration terminates with a partial result.
type rowSink interface {
	// Begin is called once with the column descriptions before any Row.
	// For command-only statements (no rows), Begin is still called with an
	// empty slice so the sink can lock in its rows-vs-command shape.
	Begin(fields []pgconn.FieldDescription) error
	// Row delivers one decoded row. Values aligns with the fields passed to
	// Begin and uses decodeTextValue's Go type mapping (int64, float64,
	// time.Time, ...).
	Row(values []any) error
	// End is called once on successful completion.
	End(ctx context.Context, commandTag string) error
	// OnError is called if iteration errors after Begin returned successfully.
	// The sink is expected to flush any in-progress output structures so
	// stdout remains well-formed. The caller still surfaces err to its caller.
	// If Begin itself errors, OnError is NOT called: sinks must not write any
	// framing before Begin succeeds.
	OnError(err error)
}

// executeOne runs a single SQL statement and streams the result through sink.
//
// ExecParams uses Postgres' extended protocol with a single unnamed statement
// and text-format result columns. There is no statement caching (no benefit
// for a one-shot CLI) and no implicit transaction wrap, so
// transaction-disallowed DDL like CREATE DATABASE works. Columns are decoded
// from their text form by decodeTextValue so all three sinks see uniform
// Go-typed input.
func executeOne(ctx context.Context, conn *pgconn.PgConn, sql string, sink rowSink) error {
	rr := conn.ExecParams(ctx, sql, nil, nil, nil, nil)

	fields := rr.FieldDescriptions()
	if err := sink.Begin(fields); err != nil {
		_, _ = rr.Close()
		return err
	}

	for rr.NextRow() {
		raw := rr.Values()
		values := make([]any, len(raw))
		for i, b := range raw {
			v, err := decodeTextValue(fields[i].DataTypeOID, b)
			if err != nil {
				_, _ = rr.Close()
				sink.OnError(err)
				return fmt.Errorf("decode row: %w", err)
			}
			values[i] = v
		}
		if err := sink.Row(values); err != nil {
			_, _ = rr.Close()
			sink.OnError(err)
			return err
		}
	}
	tag, err := rr.Close()
	if err != nil {
		sink.OnError(err)
		return fmt.Errorf("query failed: %w", err)
	}

	return sink.End(ctx, tag.String())
}

package postgrescmd

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	// Begin and uses pgx's Go type mapping (int64, float64, time.Time, ...).
	Row(values []any) error
	// End is called once on successful completion.
	End(commandTag string) error
	// OnError is called if iteration errors after Begin returned. The sink
	// is expected to flush any in-progress output structures so stdout
	// remains well-formed. The caller still surfaces err to its caller.
	OnError(err error)
}

// executeOne runs a single SQL statement and streams the result through sink.
//
// We pass QueryExecModeExec explicitly (not the pgx default
// QueryExecModeCacheStatement) for two reasons:
//
//  1. Statement caching has no benefit for a one-shot CLI: the connection is
//     closed at the end of the command, so the cached prepared statement
//     never gets reused.
//  2. Exec mode uses Postgres' extended-protocol "exec" path with text-format
//     result columns, which keeps the canonical-Postgres-text rendering for
//     --output text and --output csv straightforward.
//
// QueryExecModeExec still uses extended protocol with a single statement and
// no implicit transaction wrap, so transaction-disallowed DDL like
// CREATE DATABASE works.
func executeOne(ctx context.Context, conn *pgx.Conn, sql string, sink rowSink) error {
	rows, err := conn.Query(ctx, sql, pgx.QueryExecModeExec)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	if err := sink.Begin(rows.FieldDescriptions()); err != nil {
		return err
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			sink.OnError(err)
			return fmt.Errorf("decode row: %w", err)
		}
		if err := sink.Row(values); err != nil {
			sink.OnError(err)
			return err
		}
	}
	if err := rows.Err(); err != nil {
		sink.OnError(err)
		return fmt.Errorf("query failed: %w", err)
	}

	return sink.End(rows.CommandTag().String())
}

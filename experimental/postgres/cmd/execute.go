package postgrescmd

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// executeOne runs a single SQL statement against an open connection and
// captures the result in a queryResult.
//
// We pass QueryExecModeExec explicitly (not the pgx default
// QueryExecModeCacheStatement) for two reasons:
//
//  1. Statement caching has no benefit for a one-shot CLI: the connection is
//     closed at the end of the command, so the cached prepared statement
//     never gets reused.
//  2. Exec mode uses Postgres' extended-protocol "exec" path with text-format
//     result columns. That makes rendering canonical-Postgres-text output
//     (PR 1) and CSV (later PR) straightforward; the cache mode defaults to
//     binary and we'd be reformatting back to text.
//
// QueryExecModeExec still uses extended protocol with a single statement and
// no implicit transaction wrap, so transaction-disallowed DDL like
// `CREATE DATABASE` works.
func executeOne(ctx context.Context, conn *pgx.Conn, sql string) (*queryResult, error) {
	rows, err := conn.Query(ctx, sql, pgx.QueryExecModeExec)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	result := &queryResult{SQL: sql}

	fields := rows.FieldDescriptions()
	if len(fields) > 0 {
		result.Columns = make([]string, len(fields))
		for i, f := range fields {
			result.Columns[i] = f.Name
		}
	}

	for rows.Next() {
		raw := rows.RawValues()
		row := make([]string, len(raw))
		for i, b := range raw {
			if b == nil {
				row[i] = "NULL"
				continue
			}
			row[i] = string(b)
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	result.CommandTag = rows.CommandTag().String()
	return result, nil
}

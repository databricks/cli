package postgrescmd

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestFormatPgError_NonPgError(t *testing.T) {
	err := errors.New("plain error")
	assert.Equal(t, "plain error", formatPgError(err))
}

func TestFormatPgError_BasicPgError(t *testing.T) {
	err := &pgconn.PgError{
		Severity: "ERROR",
		Code:     "42601",
		Message:  `syntax error at or near "FRO"`,
	}
	assert.Equal(t,
		`ERROR: syntax error at or near "FRO" (SQLSTATE 42601)`,
		formatPgError(err),
	)
}

func TestFormatPgError_WithDetailAndHint(t *testing.T) {
	err := &pgconn.PgError{
		Severity: "ERROR",
		Code:     "42601",
		Message:  `syntax error at or near "FRO"`,
		Hint:     `Did you mean "FROM"?`,
		Detail:   "more context",
	}
	got := formatPgError(err)
	assert.Contains(t, got, "ERROR:")
	assert.Contains(t, got, "(SQLSTATE 42601)")
	assert.Contains(t, got, "DETAIL: more context")
	assert.Contains(t, got, `HINT:   Did you mean "FROM"?`)
}

func TestFormatPgError_WrappedPgError(t *testing.T) {
	pg := &pgconn.PgError{Code: "42501", Message: "permission denied"}
	wrapped := errors.New("query failed: " + pg.Error())
	// Plain error doesn't unwrap; falls through to err.Error.
	assert.Contains(t, formatPgError(wrapped), "permission denied")
}

package postgrescmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// formatPgError renders an error in a friendlier form when it's a Postgres
// server-side error. *pgconn.PgError exposes Code, Severity, Message, Detail,
// Hint, and Position; the plain text form attaches what's set so users see
// SQLSTATE plus any hint upstream included.
//
// For non-PgError values, returns err.Error() unchanged so the caller can
// surface it directly. The richer LINE+caret rendering is out of scope for
// this PR; we stick with the plain shape for now.
func formatPgError(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err.Error()
	}

	var sb strings.Builder
	if pgErr.Severity != "" {
		fmt.Fprintf(&sb, "%s: ", pgErr.Severity)
	} else {
		sb.WriteString("ERROR: ")
	}
	sb.WriteString(pgErr.Message)
	if pgErr.Code != "" {
		fmt.Fprintf(&sb, " (SQLSTATE %s)", pgErr.Code)
	}
	if pgErr.Detail != "" {
		fmt.Fprintf(&sb, "\nDETAIL: %s", pgErr.Detail)
	}
	if pgErr.Hint != "" {
		fmt.Fprintf(&sb, "\nHINT:   %s", pgErr.Hint)
	}
	return sb.String()
}

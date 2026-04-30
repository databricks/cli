package postgrescmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// defaultConnectTimeout is the dial timeout for a single connect attempt.
// Lakebase autoscaling endpoints can be cold-starting; Postgres' own dial
// keeps trying within this window before giving up.
const defaultConnectTimeout = 120 * time.Second

// connectConfig collects everything pgx needs to dial Postgres. Kept as a
// struct rather than passed through positional args because the pgx config
// has many fields and the call sites differ between code paths (production
// vs unit tests stubbing connectFunc).
type connectConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	ConnectTimeout time.Duration
}

// retryConfig controls connect retry on idle/waking endpoints. MaxAttempts is
// the total number of attempts: 1 means no retry, 3 means up to two retries
// with backoff between. We use the count-of-attempts reading rather than
// count-of-retries to match libs/psql.RetryConfig.MaxRetries semantics, so
// behavior stays consistent across the two commands sharing a flag name.
type retryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// connectFunc is a seam for unit tests: production wires pgx.ConnectConfig,
// tests inject failures (DNS, auth, ctx-cancel mid-connect). We deliberately
// do not wrap *pgx.Conn behind an interface for query execution; that surface
// is exercised by integration tests against real Lakebase endpoints.
type connectFunc func(ctx context.Context, cfg *pgx.ConnConfig) (*pgx.Conn, error)

// buildPgxConfig parses a base DSN to inherit pgx's TLS shape, then patches
// in the resolved values. The DSN-then-patch pattern is the recommended way
// to configure pgx for `sslmode=require` because building a pgx.ConnConfig
// by hand omits internal fields that the parser sets.
func buildPgxConfig(c connectConfig) (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig("postgresql:///?sslmode=require")
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}
	cfg.Host = c.Host
	cfg.Port = uint16(c.Port)
	cfg.User = c.Username
	cfg.Password = c.Password
	cfg.Database = c.Database
	cfg.ConnectTimeout = c.ConnectTimeout
	return cfg, nil
}

// connectWithRetry dials Postgres, retrying on connect-time errors that
// indicate the endpoint is asleep or in the middle of a wake-up. Errors that
// cannot be improved by retrying (auth failures, permission errors,
// post-query errors) are returned immediately.
func connectWithRetry(ctx context.Context, cfg *pgx.ConnConfig, rc retryConfig, dial connectFunc) (*pgx.Conn, error) {
	if rc.MaxAttempts < 1 {
		rc.MaxAttempts = 1
	}

	delay := rc.InitialDelay
	var lastErr error

	for attempt := 1; attempt <= rc.MaxAttempts; attempt++ {
		if attempt > 1 {
			cmdio.LogString(ctx, fmt.Sprintf("Connection attempt %d/%d failed, retrying in %v...", attempt-1, rc.MaxAttempts, delay))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			if rc.MaxDelay > 0 {
				delay = min(delay*2, rc.MaxDelay)
			}
		}

		conn, err := dial(ctx, cfg)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		if !isRetryableConnectError(err) {
			return nil, err
		}
		log.Debugf(ctx, "retryable connect error on attempt %d: %v", attempt, err)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", rc.MaxAttempts, lastErr)
}

// isRetryableConnectError classifies whether an error from the connect path
// is a transient "endpoint asleep / cold-starting" failure.
//
// Retryable:
//   - net.OpError with Op == "dial" (DNS resolution, TCP connect refused,
//     host unreachable). The "endpoint asleep" cases.
//   - pgconn.ConnectError that wraps a retryable network error.
//   - Postgres connection-establishment SQLSTATE codes (08xxx). Lakebase
//     emits these during cold-start.
//
// Not retryable: auth errors (28xxx), permission errors (42501),
// context cancellation/deadlines, anything after Query has been issued
// (caller never passes that to us; we only run before Query).
func isRetryableConnectError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 08xxx is the connection_exception class.
		if len(pgErr.Code) == 5 && pgErr.Code[:2] == "08" {
			return true
		}
		return false
	}

	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return isRetryableConnectError(connectErr.Unwrap())
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Op == "dial"
	}

	return false
}

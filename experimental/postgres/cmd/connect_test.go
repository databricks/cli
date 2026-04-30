package postgrescmd

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCtx(t *testing.T) context.Context {
	return cmdio.MockDiscard(t.Context())
}

func TestIsRetryableConnectError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "dial error",
			err:  &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "non-dial net.OpError",
			err:  &net.OpError{Op: "read", Err: errors.New("oops")},
			want: false,
		},
		{
			name: "08006 connection failure",
			err:  &pgconn.PgError{Code: "08006", Message: "connection failure"},
			want: true,
		},
		{
			name: "08001 cannot establish",
			err:  &pgconn.PgError{Code: "08001", Message: "sqlclient unable to establish sqlconnection"},
			want: true,
		},
		{
			name: "28000 invalid auth",
			err:  &pgconn.PgError{Code: "28000", Message: "invalid authorization specification"},
			want: false,
		},
		{
			name: "28P01 invalid password",
			err:  &pgconn.PgError{Code: "28P01", Message: "invalid password"},
			want: false,
		},
		{
			name: "42501 insufficient privilege",
			err:  &pgconn.PgError{Code: "42501", Message: "permission denied"},
			want: false,
		},
		{
			name: "context cancelled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "nil error never retryable",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRetryableConnectError(tc.err))
		})
	}
}

func TestConnectWithRetry_RespectsMaxAttempts(t *testing.T) {
	ctx := testCtx(t)
	calls := 0
	dialErr := &pgconn.PgError{Code: "08006"}
	dial := func(ctx context.Context, cfg *pgx.ConnConfig) (*pgx.Conn, error) {
		calls++
		return nil, dialErr
	}
	cfg := &pgx.ConnConfig{}
	rc := retryConfig{MaxAttempts: 3, InitialDelay: 0, MaxDelay: 0}

	_, err := connectWithRetry(ctx, cfg, rc, dial)
	require.Error(t, err)
	assert.Equal(t, 3, calls, "expected 3 attempts (1 initial + 2 retries)")
}

func TestConnectWithRetry_StopsOnNonRetryable(t *testing.T) {
	ctx := testCtx(t)
	calls := 0
	authErr := &pgconn.PgError{Code: "28P01"}
	dial := func(ctx context.Context, cfg *pgx.ConnConfig) (*pgx.Conn, error) {
		calls++
		return nil, authErr
	}
	cfg := &pgx.ConnConfig{}
	rc := retryConfig{MaxAttempts: 3, InitialDelay: 0}

	_, err := connectWithRetry(ctx, cfg, rc, dial)
	require.Error(t, err)
	assert.Equal(t, 1, calls, "auth errors should not retry")
}

func TestConnectWithRetry_ZeroMaxAttemptsTreatedAsOne(t *testing.T) {
	ctx := testCtx(t)
	calls := 0
	dial := func(ctx context.Context, cfg *pgx.ConnConfig) (*pgx.Conn, error) {
		calls++
		return nil, errors.New("nope")
	}
	cfg := &pgx.ConnConfig{}
	rc := retryConfig{MaxAttempts: 0, InitialDelay: time.Millisecond}

	_, err := connectWithRetry(ctx, cfg, rc, dial)
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestBuildPgxConfig(t *testing.T) {
	cfg, err := buildPgxConfig(connectConfig{
		Host:           "host.example.com",
		Port:           5432,
		Username:       "user",
		Password:       "secret",
		Database:       "db",
		ConnectTimeout: 30 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, "host.example.com", cfg.Host)
	assert.Equal(t, uint16(5432), cfg.Port)
	assert.Equal(t, "user", cfg.User)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, "db", cfg.Database)
	assert.Equal(t, 30*time.Second, cfg.ConnectTimeout)
}

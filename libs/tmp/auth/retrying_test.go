package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/ops"
)

type mockRetrier struct {
	fn func(error) (time.Duration, bool)
}

func (m *mockRetrier) IsRetriable(err error) (time.Duration, bool) {
	return m.fn(err)
}

func TestRetryingTokenProvider_PlainErrorNotRetried(t *testing.T) {
	wantErr := errors.New("boom")
	calls := 0
	tp := TokenProviderFn(func(ctx context.Context) (*Token, error) {
		calls++
		return nil, wantErr
	})

	got := NewRetryingTokenProvider(tp)
	_, err := got.Token(context.Background())

	if !errors.Is(err, wantErr) {
		t.Errorf("Token() error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryingTokenProvider_RetriesUntilSuccess(t *testing.T) {
	retriable := errors.New("retriable")
	wantToken := &Token{Value: "ok"}
	calls := 0
	tp := TokenProviderFn(func(ctx context.Context) (*Token, error) {
		calls++
		if calls < 3 {
			return nil, retriable
		}
		return wantToken, nil
	})

	retrier := &mockRetrier{fn: func(err error) (time.Duration, bool) {
		return 0, errors.Is(err, retriable)
	}}

	got := NewRetryingTokenProvider(tp,
		ops.WithRetrier(func() ops.Retrier { return retrier }),
	)
	gotToken, err := got.Token(context.Background())

	if err != nil {
		t.Fatalf("Token() error = %v, want nil", err)
	}
	if gotToken != wantToken {
		t.Errorf("Token() = %v, want %v", gotToken, wantToken)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryingTokenProvider_NonRetriableErrorPropagates(t *testing.T) {
	retriable := errors.New("retriable")
	nonRetriable := errors.New("non-retriable")
	calls := 0
	tp := TokenProviderFn(func(ctx context.Context) (*Token, error) {
		calls++
		return nil, nonRetriable
	})

	retrier := &mockRetrier{fn: func(err error) (time.Duration, bool) {
		return 0, errors.Is(err, retriable)
	}}

	got := NewRetryingTokenProvider(tp,
		ops.WithRetrier(func() ops.Retrier { return retrier }),
	)
	_, err := got.Token(context.Background())

	if !errors.Is(err, nonRetriable) {
		t.Errorf("Token() error = %v, want %v", err, nonRetriable)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryingTokenProvider_ContextCancellation(t *testing.T) {
	retriable := errors.New("retriable")
	tp := TokenProviderFn(func(ctx context.Context) (*Token, error) {
		return nil, retriable
	})

	retrier := &mockRetrier{fn: func(err error) (time.Duration, bool) {
		return 1 * time.Hour, errors.Is(err, retriable)
	}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := NewRetryingTokenProvider(tp,
		ops.WithRetrier(func() ops.Retrier { return retrier }),
	)
	_, err := got.Token(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Token() error = %v, want context.Canceled", err)
	}
}

// TestRetryingTokenProvider_DefaultRetrier verifies the default retrier
// retries on an APIError with a retriable HTTP status. The bridge from
// oauth2.RetrieveError to APIError happens inside fetchToken; at this layer
// the provider only deals with apierr taxonomy.
func TestRetryingTokenProvider_DefaultRetrier(t *testing.T) {
	transient := apierr.FromHTTPError(http.StatusServiceUnavailable, nil, nil)
	wantToken := &Token{Value: "ok"}
	calls := 0
	tp := TokenProviderFn(func(ctx context.Context) (*Token, error) {
		calls++
		if calls < 3 {
			return nil, transient
		}
		return wantToken, nil
	})

	got := NewRetryingTokenProvider(tp)
	gotToken, err := got.Token(context.Background())

	if err != nil {
		t.Fatalf("Token() error = %v, want nil", err)
	}
	if gotToken != wantToken {
		t.Errorf("Token() = %v, want %v", gotToken, wantToken)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

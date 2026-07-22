package auth

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func waitForAsyncRefreshToComplete(t *testing.T, cts *cachedTokenProvider) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cts.mu.Lock()
		isRefreshing := cts.isRefreshing
		cts.mu.Unlock()

		if !isRefreshing {
			return
		}

		time.Sleep(1 * time.Millisecond)
	}

	t.Fatal("timed out waiting for async refresh to complete")
}

func TestNewCachedTokenProvider_noCaching(t *testing.T) {
	want := &cachedTokenProvider{}
	got := NewCachedTokenProvider(want, nil)
	if got != want {
		t.Errorf("NewCachedTokenProvider() = %v, want %v", got, want)
	}
}

func TestNewCachedTokenProvider_default(t *testing.T) {
	ts := TokenProviderFn(func(_ context.Context) (*Token, error) {
		return nil, nil
	})

	got, ok := NewCachedTokenProvider(ts).(*cachedTokenProvider)
	if !ok {
		t.Fatalf("NewCachedTokenProvider() = %T, want *cachedTokenProvider", got)
	}

	if !got.nextAsyncRefresh.IsZero() {
		t.Errorf("NewCachedTokenProvider() nextAsyncRefresh = %v, want zero value", got.nextAsyncRefresh)
	}
	if got.disableAsync != false {
		t.Errorf("NewCachedTokenProvider() disableAsync = %v, want %v", got.disableAsync, false)
	}
	if got.cachedToken != nil {
		t.Errorf("NewCachedTokenProvider() cachedToken = %v, want nil", got.cachedToken)
	}
}

func TestNewCachedTokenProvider_options(t *testing.T) {
	ts := TokenProviderFn(func(_ context.Context) (*Token, error) {
		return nil, nil
	})

	wantDisableAsync := false
	wantCachedToken := &Token{Expiry: time.Unix(42, 0)}

	opts := []Option{
		WithAsyncRefresh(!wantDisableAsync),
		WithCachedToken(wantCachedToken),
	}

	got, ok := NewCachedTokenProvider(ts, opts...).(*cachedTokenProvider)
	if !ok {
		t.Fatalf("NewCachedTokenProvider() = %T, want *cachedTokenProvider", got)
	}

	if got.disableAsync != wantDisableAsync {
		t.Errorf("NewCachedTokenProvider(): disableAsync = %v, want %v", got.disableAsync, wantDisableAsync)
	}
	if got.cachedToken != wantCachedToken {
		t.Errorf("NewCachedTokenProvider(): cachedToken = %v, want %v", got.cachedToken, wantCachedToken)
	}
}

func TestCachedTokenProvider_updateNextAsyncRefresh(t *testing.T) {
	now := time.Unix(1337, 0)

	testCases := []struct {
		name             string
		tokenTTL         time.Duration
		wantAllowedAfter time.Duration
	}{
		{
			name:             "standard OAuth token with 60-minute TTL",
			tokenTTL:         60 * time.Minute,
			wantAllowedAfter: 40 * time.Minute,
		},
		{
			name:             "short-lived token with 10-minute TTL",
			tokenTTL:         10 * time.Minute,
			wantAllowedAfter: 5 * time.Minute,
		},
		{
			name:             "very short token with 90-second TTL",
			tokenTTL:         90 * time.Second,
			wantAllowedAfter: 45 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cts := &cachedTokenProvider{
				cachedToken: &Token{
					Expiry: now.Add(tc.tokenTTL),
				},
				timeNow: func() time.Time { return now },
			}

			cts.updateNextAsyncRefresh()

			want := now.Add(tc.wantAllowedAfter)
			if cts.nextAsyncRefresh != want {
				t.Errorf("nextAsyncRefresh = %v, want %v", cts.nextAsyncRefresh, want)
			}
		})
	}
}

func TestCachedTokenProvider_tokenExpired(t *testing.T) {
	now := time.Unix(1337, 0) // mock value for time.Now()

	testCases := []struct {
		name  string
		token *Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  true,
		},
		{
			name: "expired token",
			token: &Token{
				Expiry: now.Add(-1 * time.Second),
			},
			want: true,
		},
		{
			name: "token expiring now",
			token: &Token{
				Expiry: now,
			},
			want: false,
		},
		{
			name: "future token",
			token: &Token{
				Expiry: now.Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "token without expiry",
			token: &Token{
				Expiry: time.Time{},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cts := &cachedTokenProvider{
				cachedToken: tc.token,
				timeNow:     func() time.Time { return now },
			}

			got := cts.tokenExpired()

			if got != tc.want {
				t.Errorf("tokenExpired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCachedTokenProvider_canRefreshAsync(t *testing.T) {
	now := time.Unix(1337, 0) // mock value for time.Now()

	testCases := []struct {
		name             string
		token            *Token
		nextAsyncRefresh time.Time
		want             bool
	}{
		{
			name: "nil token",
			want: false,
		},
		{
			name: "token without expiry",
			token: &Token{
				Expiry: time.Time{},
			},
			want: false,
		},
		{
			name: "before async refresh window",
			token: &Token{
				Expiry: now.Add(1 * time.Hour),
			},
			nextAsyncRefresh: now.Add(1 * time.Minute),
			want:             false,
		},
		{
			name: "exactly at async refresh boundary",
			token: &Token{
				Expiry: now.Add(1 * time.Hour),
			},
			nextAsyncRefresh: now,
			want:             false,
		},
		{
			name: "after async refresh boundary",
			token: &Token{
				Expiry: now.Add(1 * time.Hour),
			},
			nextAsyncRefresh: now.Add(-1 * time.Second),
			want:             true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cts := &cachedTokenProvider{
				cachedToken:      tc.token,
				nextAsyncRefresh: tc.nextAsyncRefresh,
				timeNow:          func() time.Time { return now },
			}

			got := cts.canRefreshAsync()

			if got != tc.want {
				t.Errorf("canRefreshAsync() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCachedTokenProvider_AsyncRefreshRetry(t *testing.T) {
	failTime := time.Unix(1337, 0)
	currentToken := &Token{
		Value:  "current-token",
		Expiry: failTime.Add(5 * time.Minute),
	}
	refreshedToken := &Token{
		Value:  "refreshed-token",
		Expiry: failTime.Add(1 * time.Hour),
	}

	testCases := []struct {
		name      string
		now       time.Time
		wantCalls int32
		wantCache *Token
	}{
		{
			name:      "no async refresh allowed during backoff",
			now:       failTime.Add(asyncRefreshRetryBackoff - 1*time.Second),
			wantCalls: 0,
			wantCache: currentToken,
		},
		{
			name:      "async refresh allowed after backoff",
			now:       failTime.Add(asyncRefreshRetryBackoff + 1*time.Second),
			wantCalls: 1,
			wantCache: refreshedToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCalls := int32(0)
			cts := &cachedTokenProvider{
				cachedToken:      currentToken,
				nextAsyncRefresh: failTime.Add(asyncRefreshRetryBackoff),
				timeNow:          func() time.Time { return tc.now },
				TokenProvider: TokenProviderFn(func(_ context.Context) (*Token, error) {
					atomic.AddInt32(&gotCalls, 1)
					return refreshedToken, nil
				}),
			}

			gotToken, err := cts.Token(context.Background())
			if err != nil {
				t.Fatalf("Token() error = %v", err)
			}
			if gotToken != currentToken {
				t.Errorf("Token() = %v, want %v", gotToken, currentToken)
			}

			waitForAsyncRefreshToComplete(t, cts)

			if got := atomic.LoadInt32(&gotCalls); got != tc.wantCalls {
				t.Fatalf("token source calls = %d, want %d", got, tc.wantCalls)
			}
			if cts.cachedToken != tc.wantCache {
				t.Errorf("cachedToken = %v, want %v", cts.cachedToken, tc.wantCache)
			}
		})
	}
}

func TestCachedTokenProvider_Token(t *testing.T) {
	now := time.Unix(1337, 0) // mock value for time.Now()
	nTokenCalls := 10         // number of goroutines calling Token()
	testCases := []struct {
		desc             string // description of the test case
		cachedToken      *Token // token cached before calling Token()
		disableAsync     bool   // whether async refreshes are disabled
		nextAsyncRefresh time.Time

		returnedToken *Token // token returned by the token source
		returnedError error  // error returned by the token source

		wantCalls int    // expected number of calls to the token source
		wantToken *Token // expected token in the cache
	}{
		{
			desc:          "[Blocking] no cached token",
			disableAsync:  true,
			returnedToken: &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:     1,
			wantToken:     &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:          "[Blocking] expired cached token",
			disableAsync:  true,
			cachedToken:   &Token{Expiry: now.Add(-1 * time.Second)},
			returnedToken: &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:     1,
			wantToken:     &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:         "[Blocking] fresh cached token",
			disableAsync: true,
			cachedToken:  &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:    0,
			wantToken:    &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:         "[Blocking] stale cached token",
			disableAsync: true,
			cachedToken:  &Token{Expiry: now.Add(1 * time.Minute)},
			wantCalls:    0,
			wantToken:    &Token{Expiry: now.Add(1 * time.Minute)},
		},
		{
			desc:          "[Blocking] refresh error",
			disableAsync:  true,
			returnedError: fmt.Errorf("test error"),
			wantCalls:     10,
		},
		{
			desc:          "[Blocking] recover from error",
			disableAsync:  true,
			cachedToken:   &Token{Expiry: now.Add(-1 * time.Minute)},
			returnedToken: &Token{Expiry: now.Add(-1 * time.Hour)},
			wantCalls:     10,
			wantToken:     &Token{Expiry: now.Add(-1 * time.Hour)},
		},
		{
			desc:          "[Async] no cached token",
			returnedToken: &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:     1,
			wantToken:     &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:          "[Async] expired cached token",
			cachedToken:   &Token{Expiry: now.Add(-1 * time.Second)},
			returnedToken: &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:     1,
			wantToken:     &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:             "[Async] fresh cached token",
			cachedToken:      &Token{Expiry: now.Add(1 * time.Hour)},
			nextAsyncRefresh: now.Add(1 * time.Minute),
			wantCalls:        0,
			wantToken:        &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:             "[Async] stale cached token",
			cachedToken:      &Token{Expiry: now.Add(1 * time.Minute)},
			nextAsyncRefresh: now.Add(-1 * time.Second),
			returnedToken:    &Token{Expiry: now.Add(1 * time.Hour)},
			wantCalls:        1,
			wantToken:        &Token{Expiry: now.Add(1 * time.Hour)},
		},
		{
			desc:             "[Async] refresh error",
			cachedToken:      &Token{Expiry: now.Add(1 * time.Minute)},
			nextAsyncRefresh: now.Add(-1 * time.Second),
			returnedError:    fmt.Errorf("test error"),
			wantCalls:        1,
			wantToken:        &Token{Expiry: now.Add(1 * time.Minute)},
		},
		{
			desc:             "[Async] stale cached token, expired token returned",
			cachedToken:      &Token{Expiry: now.Add(1 * time.Minute)},
			nextAsyncRefresh: now.Add(-1 * time.Second),
			returnedToken:    &Token{Expiry: now.Add(-1 * time.Second)},
			wantCalls:        1,
			wantToken:        &Token{Expiry: now.Add(-1 * time.Second)},
		},
		{
			desc:          "[Async] recover from error",
			cachedToken:   &Token{Expiry: now.Add(-1 * time.Minute)},
			returnedToken: &Token{Expiry: now.Add(-1 * time.Hour)},
			wantCalls:     10,
			wantToken:     &Token{Expiry: now.Add(-1 * time.Hour)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			gotCalls := int32(0)
			cts := &cachedTokenProvider{
				disableAsync:     tc.disableAsync,
				cachedToken:      tc.cachedToken,
				nextAsyncRefresh: tc.nextAsyncRefresh,
				timeNow:          func() time.Time { return now },
				TokenProvider: TokenProviderFn(func(_ context.Context) (*Token, error) {
					atomic.AddInt32(&gotCalls, 1)
					time.Sleep(10 * time.Millisecond)
					return tc.returnedToken, tc.returnedError
				}),
			}

			wg := sync.WaitGroup{}
			for range nTokenCalls {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, _ = cts.Token(context.Background())
				}()
			}

			wg.Wait()

			waitForAsyncRefreshToComplete(t, cts)

			if got := int(atomic.LoadInt32(&gotCalls)); got != tc.wantCalls {
				t.Errorf("want %d calls to cts.TokenProvider.Token(), got %d", tc.wantCalls, got)
			}
			if !reflect.DeepEqual(tc.wantToken, cts.cachedToken) {
				t.Errorf("want cached token %v, got %v", tc.wantToken, cts.cachedToken)
			}
		})
	}
}

func TestCachedTokenProvider_BlockingRefreshUpdatesNextAsyncRefresh(t *testing.T) {
	now := time.Unix(1337, 0)
	refreshedToken := &Token{Expiry: now.Add(1 * time.Hour)}

	cts := &cachedTokenProvider{
		disableAsync:     true,
		nextAsyncRefresh: now.Add(-2 * time.Minute),
		timeNow:          func() time.Time { return now },
		cachedToken:      &Token{Expiry: now.Add(-1 * time.Second)}, // expired
		TokenProvider: TokenProviderFn(func(_ context.Context) (*Token, error) {
			return refreshedToken, nil
		}),
	}

	_, err := cts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	want := refreshedToken.Expiry.Add(-maxAsyncRefreshLeadTime)
	if cts.nextAsyncRefresh != want {
		t.Errorf("nextAsyncRefresh = %v, want %v", cts.nextAsyncRefresh, want)
	}
}

func TestCachedTokenProvider_AsyncRefreshSkipsOlderToken(t *testing.T) {
	now := time.Unix(1337, 0)
	cachedToken := &Token{
		Value:  "fresh-from-blocking",
		Expiry: now.Add(1 * time.Hour),
	}
	olderToken := &Token{
		Value:  "stale-from-async",
		Expiry: now.Add(30 * time.Minute),
	}

	cts := &cachedTokenProvider{
		cachedToken:      &Token{Value: "original", Expiry: now.Add(2 * time.Minute)},
		nextAsyncRefresh: now.Add(-1 * time.Second),
		timeNow:          func() time.Time { return now },
	}
	cts.TokenProvider = TokenProviderFn(func(_ context.Context) (*Token, error) {
		// Simulate blocking refresh completing first by swapping in the
		// fresh token and bumping the generation before this goroutine returns.
		cts.mu.Lock()
		cts.cachedToken = cachedToken
		cts.tokenGeneration++
		cts.mu.Unlock()
		return olderToken, nil
	})

	_, err := cts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	waitForAsyncRefreshToComplete(t, cts)

	if cts.cachedToken != cachedToken {
		t.Errorf("cachedToken = %v, want %v (fresher token from blocking refresh)", cts.cachedToken, cachedToken)
	}
}

func TestCachedTokenProvider_AsyncRefreshAcceptsShorterTTL(t *testing.T) {
	now := time.Unix(1337, 0)
	originalToken := &Token{
		Value:  "original",
		Expiry: now.Add(5 * time.Minute),
	}
	shorterTTLToken := &Token{
		Value:  "shorter-ttl",
		Expiry: now.Add(2 * time.Minute),
	}

	cts := &cachedTokenProvider{
		cachedToken:      originalToken,
		nextAsyncRefresh: now.Add(-1 * time.Second),
		timeNow:          func() time.Time { return now },
		TokenProvider: TokenProviderFn(func(_ context.Context) (*Token, error) {
			return shorterTTLToken, nil
		}),
	}

	gotToken, err := cts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if gotToken != originalToken {
		t.Errorf("Token() = %v, want %v (stale token returned immediately)", gotToken, originalToken)
	}

	waitForAsyncRefreshToComplete(t, cts)

	if cts.cachedToken != shorterTTLToken {
		t.Errorf("cachedToken = %v, want %v (shorter-TTL token should be accepted)", cts.cachedToken, shorterTTLToken)
	}
}

func TestCachedTokenProvider_LateAsyncFailureDoesNotBackoff(t *testing.T) {
	now := time.Unix(1337, 0)
	staleToken := &Token{
		Value:  "stale",
		Expiry: now.Add(5 * time.Minute),
	}
	blockingResult := &Token{
		Value:  "blocking-new",
		Expiry: now.Add(1 * time.Hour),
	}

	cts := &cachedTokenProvider{
		cachedToken:      staleToken,
		nextAsyncRefresh: now.Add(-1 * time.Second),
		timeNow:          func() time.Time { return now },
	}
	cts.TokenProvider = TokenProviderFn(func(_ context.Context) (*Token, error) {
		// Simulate a blocking refresh completing first: update the token
		// and bump the generation before this async goroutine returns an error.
		cts.mu.Lock()
		cts.cachedToken = blockingResult
		cts.tokenGeneration++
		cts.updateNextAsyncRefresh()
		cts.mu.Unlock()
		return nil, fmt.Errorf("late async failure")
	})

	_, err := cts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	waitForAsyncRefreshToComplete(t, cts)

	// The late async failure must not have pushed nextAsyncRefresh forward
	// because the token generation changed while it was in flight.
	wantNextAsync := blockingResult.Expiry.Add(-maxAsyncRefreshLeadTime)
	if cts.nextAsyncRefresh != wantNextAsync {
		t.Errorf("nextAsyncRefresh = %v, want %v (late failure should not apply backoff to newer generation)",
			cts.nextAsyncRefresh, wantNextAsync)
	}
	if cts.cachedToken != blockingResult {
		t.Errorf("cachedToken = %v, want %v", cts.cachedToken, blockingResult)
	}
}

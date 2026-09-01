package fuse

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staticToken(value string) TokenFunc {
	return func(ctx context.Context) (string, error) {
		return value, nil
	}
}

// stopAndWaitForRevoke cancels the registration and waits for the revoke to land,
// so the stand-in daemons are not torn down from under it.
func stopAndWaitForRevoke(t *testing.T, cancel context.CancelFunc, daemons ...*daemon) {
	t.Helper()
	counts := make([]int, len(daemons))
	for i, d := range daemons {
		counts[i] = d.count()
	}
	cancel()
	for i, d := range daemons {
		require.Eventually(t, func() bool { return d.count() > counts[i] }, waitFor, tick)
	}
}

func TestKeepRegisteredRegistersBeforeItReturns(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	ctx, cancel := context.WithCancel(t.Context())
	r := Registration{PID: 4242, PidNamespaceID: 7, StartTime: 99}
	defer func() { stopAndWaitForRevoke(t, cancel, wsfs, ucFuse) }()

	// The caller goes straight on to touch /Workspace, so the first registration
	// has to have landed by the time this returns.
	require.NoError(t, KeepRegistered(ctx, client, r, staticToken("token"), "42"))

	require.Equal(t, 1, wsfs.count())
	require.Equal(t, 1, ucFuse.count())
	body := wsfs.at(0).body
	assert.Equal(t, "token", body.APIToken)
	assert.Equal(t, uint64(99), body.ProcStartTime)
	assert.Equal(t, map[string]string{"userId": "42"}, body.AdditionalTags)
}

func TestKeepRegisteredReportsAFirstRegistrationThatFailed(t *testing.T) {
	client, _, _ := newTestClient(t)
	tokenErr := errors.New("no credentials")

	err := KeepRegistered(t.Context(), client, Registration{PID: 1, PidNamespaceID: 2, StartTime: 3},
		func(ctx context.Context) (string, error) { return "", tokenErr }, "")

	// The server only logs a warning and carries on, so the reason has to survive.
	assert.ErrorIs(t, err, tokenErr)
	assert.ErrorContains(t, err, "failed to get a token to register")
}

func TestKeepRegisteredReportsDaemonsThatRefused(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	wsfs.failWith(http.StatusBadRequest)
	ucFuse.failWith(http.StatusBadRequest)

	err := KeepRegistered(t.Context(), client, Registration{PID: 1, PidNamespaceID: 2, StartTime: 3},
		staticToken("token"), "")
	assert.ErrorContains(t, err, "failed to register pid=1 ns=2 startTime=3 with the FUSE daemons")
}

func TestKeepRegisteredRevokesWhenTheServerStops(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	ctx, cancel := context.WithCancel(t.Context())
	r := Registration{PID: 4242, PidNamespaceID: 7, StartTime: 99}

	require.NoError(t, KeepRegistered(ctx, client, r, staticToken("token"), ""))
	cancel()

	// The revoke runs on a context of its own, because the one it is triggered by
	// is already cancelled.
	require.Eventually(t, func() bool { return wsfs.count() == 2 }, waitFor, tick)
	require.Eventually(t, func() bool { return ucFuse.count() == 2 }, waitFor, tick)

	revoke := wsfs.at(1)
	assert.Empty(t, revoke.body.APIToken, "an empty token is what the daemons read as a revoke")
	assert.Zero(t, revoke.body.ProcStartTime)
	assert.Equal(t, "/api/1/pid/4242", revoke.path)
}

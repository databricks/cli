package sandbox

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const waitTestSandboxID = "test-id"

type fakeRunWaiter struct {
	statuses []string
	getErr   error
	startErr error
	polls    int
	starts   []time.Time
}

func (waiter *fakeRunWaiter) get(_ context.Context, id string) (*sandboxEntry, error) {
	if waiter.getErr != nil {
		return nil, waiter.getErr
	}
	status := waiter.statuses[min(waiter.polls, len(waiter.statuses)-1)]
	waiter.polls++
	return &sandboxEntry{SandboxID: id, Status: status}, nil
}

func (waiter *fakeRunWaiter) start(_ context.Context, id string) (*sandboxEntry, error) {
	waiter.starts = append(waiter.starts, time.Now())
	if waiter.startErr != nil {
		return nil, waiter.startErr
	}
	return &sandboxEntry{SandboxID: id, Status: "Creating"}, nil
}

func TestWaitForRunning(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		statuses   []string
		startErr   error
		wantErr    string
		wantStarts int
	}{
		{
			name:     "already running",
			statuses: []string{"Running"},
		},
		{
			name:     "creating",
			statuses: []string{"Creating", "Creating", "Running"},
		},
		{
			name:     "transient stopped",
			statuses: []string{"Stopped", "Creating", "Running"},
		},
		{
			name:       "stopped retries start",
			statuses:   append(slices.Repeat([]string{"Stopped"}, 9), "Creating", "Running"),
			wantStarts: 1,
		},
		{
			name:       "case insensitive statuses",
			statuses:   append(slices.Repeat([]string{"sToPpEd"}, 9), "rUnNiNg"),
			wantStarts: 1,
		},
		{
			name:       "retries are throttled",
			statuses:   append(slices.Repeat([]string{"Stopped"}, 18), "Running"),
			wantStarts: 2,
		},
		{
			name:       "retry error is tolerated",
			statuses:   append(slices.Repeat([]string{"Stopped"}, 9), "Creating", "Running"),
			startErr:   errors.New("currently stopping"),
			wantStarts: 1,
		},
		{
			name:     "failed is terminal",
			statuses: []string{"Failed"},
			wantErr:  `sandbox test-id reached unexpected state "Failed" while starting`,
		},
		{
			name:     "terminated is terminal",
			statuses: []string{"Terminated"},
			wantErr:  `sandbox test-id reached unexpected state "Terminated" while starting`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx := cmdio.MockDiscard(t.Context())
				progress := spin(ctx, "")
				defer progress.Close()
				waiter := &fakeRunWaiter{statuses: testCase.statuses, startErr: testCase.startErr}
				started := time.Now()

				got, err := waitForRunning(ctx, waiter, progress, waitTestSandboxID)
				if testCase.wantErr != "" {
					require.EqualError(t, err, testCase.wantErr)
					assert.Nil(t, got)
				} else {
					require.NoError(t, err)
					assert.Equal(t, waitTestSandboxID, got.SandboxID)
					assert.Equal(t, testCase.statuses[len(testCase.statuses)-1], got.Status)
				}
				assert.Equal(t, len(testCase.statuses), waiter.polls)
				assert.Len(t, waiter.starts, testCase.wantStarts)
				lastStart := started
				for _, retried := range waiter.starts {
					assert.GreaterOrEqual(t, retried.Sub(lastStart), startRenudgeInterval)
					lastStart = retried
				}
			})
		})
	}
}

func TestWaitForRunningTimeout(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"Creating", "Stopped"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx := cmdio.MockDiscard(t.Context())
				progress := spin(ctx, "")
				defer progress.Close()
				waiter := &fakeRunWaiter{statuses: []string{status}}
				started := time.Now()

				got, err := waitForRunning(ctx, waiter, progress, waitTestSandboxID)
				require.ErrorContains(t, err, "did not reach Running within")
				assert.ErrorContains(t, err, "last seen "+status)
				assert.Nil(t, got)
				assert.GreaterOrEqual(t, time.Since(started), startWaitTimeout)
				assert.LessOrEqual(t, time.Since(started), startWaitTimeout+startPollInterval)
				if status == "Stopped" {
					require.NotEmpty(t, waiter.starts)
					lastStart := started
					for _, retried := range waiter.starts {
						assert.GreaterOrEqual(t, retried.Sub(lastStart), startRenudgeInterval)
						assert.LessOrEqual(t, retried.Sub(started), startWaitTimeout)
						lastStart = retried
					}
				} else {
					assert.Empty(t, waiter.starts)
				}
			})
		})
	}
}

func TestWaitForRunningPollingError(t *testing.T) {
	t.Parallel()

	ctx := cmdio.MockDiscard(t.Context())
	progress := spin(ctx, "")
	defer progress.Close()
	pollErr := errors.New("sandbox not found")
	waiter := &fakeRunWaiter{getErr: pollErr}

	got, err := waitForRunning(ctx, waiter, progress, waitTestSandboxID)
	require.ErrorIs(t, err, pollErr)
	assert.ErrorContains(t, err, "polling status of "+waitTestSandboxID)
	assert.Nil(t, got)
	assert.Empty(t, waiter.starts)
}

func TestWaitForRunningCancellation(t *testing.T) {
	t.Parallel()

	for _, cancelAfter := range []time.Duration{0, 5 * time.Second, 17 * time.Second} {
		t.Run(cancelAfter.String(), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
				defer cancel()
				if cancelAfter == 0 {
					cancel()
				} else {
					timer := time.AfterFunc(cancelAfter, cancel)
					defer timer.Stop()
				}
				progress := spin(ctx, "")
				defer progress.Close()
				waiter := &fakeRunWaiter{statuses: []string{"Stopped"}}
				started := time.Now()

				got, err := waitForRunning(ctx, waiter, progress, waitTestSandboxID)
				assert.ErrorIs(t, err, context.Canceled)
				assert.Nil(t, got)
				assert.Equal(t, cancelAfter, time.Since(started))
			})
		})
	}
}

func TestWaitForStoppedDoesNotRestart(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		statuses []string
		wantErr  string
	}{
		{
			name:     "already stopped",
			statuses: []string{"Stopped"},
		},
		{
			name:     "stopping",
			statuses: []string{"Stopping", "Stopped"},
		},
		{
			name:     "failed is terminal",
			statuses: []string{"Failed"},
			wantErr:  `sandbox test-id reached unexpected state "Failed" while stopping`,
		},
		{
			name:     "terminated is terminal",
			statuses: []string{"Terminated"},
			wantErr:  `sandbox test-id reached unexpected state "Terminated" while stopping`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx := cmdio.MockDiscard(t.Context())
				progress := spin(ctx, "")
				defer progress.Close()
				waiter := &fakeRunWaiter{statuses: testCase.statuses}

				got, err := waitForStopped(ctx, waiter, progress, waitTestSandboxID)
				if testCase.wantErr != "" {
					require.EqualError(t, err, testCase.wantErr)
					assert.Nil(t, got)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "Stopped", got.Status)
				}
				assert.Empty(t, waiter.starts)
			})
		})
	}
}

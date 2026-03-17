package acceptance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTestPhase(t *testing.T) {
	tests := []struct {
		name    string
		phase   int
		wantErr string
	}{
		{
			name:  "phase zero",
			phase: 0,
		},
		{
			name:  "phase one",
			phase: 1,
		},
		{
			name:    "negative phase",
			phase:   -1,
			wantErr: "Phase must be 0 or 1, got -1",
		},
		{
			name:    "phase two",
			phase:   2,
			wantErr: "Phase must be 0 or 1, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestPhase(tt.phase)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestPhaseSemaphoreWaitsForAllPhaseZeroTests(t *testing.T) {
	semaphore := newPhaseSemaphore()
	semaphore.Add()
	semaphore.Add()
	gate := semaphore.Seal()

	released := make(chan struct{})
	go func() {
		<-gate
		close(released)
	}()

	select {
	case <-released:
		t.Fatal("phase 1 should stay blocked until phase 0 completes")
	case <-time.After(20 * time.Millisecond):
	}

	semaphore.Done()

	select {
	case <-released:
		t.Fatal("phase 1 should stay blocked until all phase 0 tests complete")
	case <-time.After(20 * time.Millisecond):
	}

	semaphore.Done()

	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("phase 1 was not released")
	}
}

func TestPhaseSemaphoreWithoutPhaseZeroTestsIsOpen(t *testing.T) {
	gate := newPhaseSemaphore().Seal()

	select {
	case <-gate:
	default:
		t.Fatal("phase 1 should be released when there are no phase 0 tests")
	}

	assert.NotNil(t, gate)
}

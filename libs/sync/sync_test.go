package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunContinuousRejectsNonPositivePollInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			s := &Sync{SyncOptions: &SyncOptions{PollInterval: interval}}
			err := s.RunContinuous(t.Context())
			assert.EqualError(t, err, "poll interval must be greater than zero")
		})
	}
}

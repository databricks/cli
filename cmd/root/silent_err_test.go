package root

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// duckStatusError has an ExitStatus method but is not an ExitStatusError, so it does not set the status.
type duckStatusError struct{}

func (duckStatusError) Error() string   { return "duck" }
func (duckStatusError) ExitStatus() int { return 7 }

func TestExitStatus(t *testing.T) {
	plain := errors.New("plain failure")
	status := &ExitStatusError{Status: 2, Err: errors.New("binding failed")}

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: 0},
		{name: "plain error", err: plain, want: 1},
		{name: "status error", err: status, want: 2},
		{name: "wrapped status error", err: fmt.Errorf("serve: %w", status), want: 2},
		{name: "joined with already printed", err: errors.Join(ErrAlreadyPrinted, status), want: 2},
		{name: "joined plain error", err: errors.Join(ErrAlreadyPrinted, plain), want: 1},
		// Only the named type sets a status: an error with an ExitStatus method of its own does not.
		{name: "method without the type", err: duckStatusError{}, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitStatus(tc.err))
		})
	}
}

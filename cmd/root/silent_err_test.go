package root

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testStatusError sets its own exit status, as mcp.BindError does.
type testStatusError struct {
	status int
	err    error
}

func (e *testStatusError) Error() string { return e.err.Error() }

func (e *testStatusError) Unwrap() error { return e.err }

func (e *testStatusError) ExitStatus() int { return e.status }

func TestExitStatus(t *testing.T) {
	plain := errors.New("plain failure")
	status := &testStatusError{status: 2, err: errors.New("binding failed")}

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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitStatus(tc.err))
		})
	}
}

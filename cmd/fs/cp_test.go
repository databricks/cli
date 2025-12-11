package fs

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

func TestCpConcurrencyValidation(t *testing.T) {
	testCases := []struct {
		concurrency int
		wantError   error
	}{
		{-1337, errInvalidConcurrency},
		{-1, errInvalidConcurrency},
		{0, errInvalidConcurrency},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("concurrency=%d", tc.concurrency), func(t *testing.T) {
			cmd := newCpCommand()
			cmd.SetArgs([]string{"src", "dst", "--concurrency", strconv.Itoa(tc.concurrency)})
			err := cmd.Execute()
			if !errors.Is(err, tc.wantError) {
				t.Errorf("expected error %v, got %v", tc.wantError, err)
			}
		})
	}
}

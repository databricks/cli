package psql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNonRetryableError(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{`FATAL:  role "user@example.com" does not exist`, true},
		{`FATAL:  database "testdb" does not exist`, true},
		{`FATAL:  password authentication failed for user "user"`, true},
		{`could not connect to server: Connection refused`, false},
		{`server closed the connection unexpectedly`, false},
		{``, false},
	}

	for _, tt := range tests {
		t.Run(tt.stderr, func(t *testing.T) {
			got := isNonRetryableError(tt.stderr)
			assert.Equal(t, tt.want, got)
		})
	}
}

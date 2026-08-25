package server

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "value with quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
		{
			name:     "value with newline",
			input:    "line1\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with carriage return",
			input:    "line1\rline2",
			expected: "line1line2",
		},
		{
			name:     "value with CRLF",
			input:    "line1\r\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with quotes and newlines",
			input:    "say \"hello\"\nworld",
			expected: `say \"hello\"world`,
		},
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\r\n",
			expected: "",
		},
		{
			name:     "backslashes",
			input:    `foo\bar\`,
			expected: `foo\\bar\\`,
		},
		{
			name:     "backslash before quote",
			input:    `foo\"bar`,
			expected: `foo\\\"bar`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeEnvValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSSHDConfigAsksTheClientToConfirmItIsStillThere(t *testing.T) {
	config := sshdConfigContent("/keys/server-private-key", "/keys/authorized_keys", `SetEnv FOO="bar"`)

	// sshd sends nothing on an idle session unless ClientAliveInterval is set, and payload is the
	// only traffic that keeps the tunnel leg past the driver proxy from being reaped (DECO-28186).
	// This is the half of the keepalive that reaches clients the CLI never configures.
	assert.Contains(t, config, "\nClientAliveInterval "+strconv.Itoa(clientAliveIntervalSeconds)+"\n")

	// The interval has to fire well inside the ~8 minute reap window, and ClientAliveCountMax
	// (OpenSSH default 3) intervals have to outlast the longest legitimate pause on a healthy
	// tunnel — the up to 30s a handover can hold the sending loop.
	assert.Less(t, clientAliveIntervalSeconds, 8*60)
	assert.Greater(t, 3*clientAliveIntervalSeconds, 30)
}

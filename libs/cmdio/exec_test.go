package cmdio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutorWithSimpleInput(t *testing.T) {
	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), "echo 'Hello'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\n", string(out))
}

func TestExecutorWithComplexInput(t *testing.T) {
	executor := NewCommandExecutor(".")
	out, err := executor.Exec(context.Background(), "echo 'Hello' && echo 'World'")
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Hello\nWorld\n", string(out))
}

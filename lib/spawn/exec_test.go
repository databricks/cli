package spawn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestExecAndPassError(t *testing.T) {
	_, err := ExecAndPassErr(context.Background(), "which", "__non_existing__")
	assert.EqualError(t, err, "exit status 1")
}

package process

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackground(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{"echo", "1"}, WithDir("/"))
	assert.NoError(t, err)
	assert.Equal(t, "1", res)
}

func TestBackgroundFails(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"ls", "/dev/null/x"})
	assert.NotNil(t, err)
}

func TestBackgroundFailsOnOption(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"ls", "/dev/null/x"}, func(c *exec.Cmd) error {
		return fmt.Errorf("nope")
	})
	assert.EqualError(t, err, "nope")
}

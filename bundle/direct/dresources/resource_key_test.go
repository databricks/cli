package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceKeyRoundTrip(t *testing.T) {
	ctx := WithResourceKey(t.Context(), "job_runs.foo")
	assert.Equal(t, "job_runs.foo", ResourceKey(ctx))
	assert.Empty(t, ResourceKey(t.Context()))
}

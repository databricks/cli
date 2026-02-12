package agent

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearAllAgentEnvVars(ctx context.Context) context.Context {
	for _, a := range knownAgents {
		ctx = env.Set(ctx, a.envVar, "")
	}
	return ctx
}

func TestDetectEachAgent(t *testing.T) {
	for _, a := range knownAgents {
		t.Run(a.product, func(t *testing.T) {
			ctx := clearAllAgentEnvVars(context.Background())
			ctx = env.Set(ctx, a.envVar, "1")

			assert.Equal(t, a.product, detect(ctx))
		})
	}
}

func TestDetectViaContext(t *testing.T) {
	ctx := clearAllAgentEnvVars(context.Background())
	ctx = env.Set(ctx, knownAgents[0].envVar, "1")

	ctx = Detect(ctx)

	assert.Equal(t, knownAgents[0].product, Product(ctx))
}

func TestDetectNoAgent(t *testing.T) {
	ctx := clearAllAgentEnvVars(context.Background())

	ctx = Detect(ctx)

	assert.Equal(t, "", Product(ctx))
}

func TestDetectMultipleAgents(t *testing.T) {
	ctx := clearAllAgentEnvVars(context.Background())
	for _, a := range knownAgents {
		ctx = env.Set(ctx, a.envVar, "1")
	}

	assert.Equal(t, "", detect(ctx))
}

func TestProductCalledBeforeDetect(t *testing.T) {
	ctx := context.Background()

	require.Panics(t, func() {
		Product(ctx)
	})
}

func TestMock(t *testing.T) {
	ctx := context.Background()
	ctx = Mock(ctx, "test-agent")

	assert.Equal(t, "test-agent", Product(ctx))
}

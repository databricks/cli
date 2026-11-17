package liteswap_test

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/liteswap"
	"github.com/databricks/cli/libs/skillsinvoked"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyComposesWithSkillsInvoked(t *testing.T) {
	ctx := env.Set(t.Context(), liteswap.TargetVariable, "dms")
	ctx = skillsinvoked.WithSkills(ctx, []string{"databricks-core"})

	tests := []struct {
		name  string
		apply func(*config.Config)
	}{
		{"liteswap then skills", func(cfg *config.Config) {
			liteswap.Apply(ctx, cfg)
			skillsinvoked.Apply(cfg)
		}},
		{"skills then liteswap", func(cfg *config.Config) {
			skillsinvoked.Apply(cfg)
			liteswap.Apply(ctx, cfg)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			tt.apply(cfg)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/api", nil)
			require.NoError(t, err)
			require.NoError(t, cfg.Headers(req))
			assert.Equal(t, "testenv://liteswap/dms", req.Header.Get("x-databricks-traffic-id"))
			assert.Equal(t, "databricks-core", req.Header.Get(skillsinvoked.Header))
		})
	}
}

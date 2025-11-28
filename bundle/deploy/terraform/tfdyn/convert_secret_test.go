package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSecret(t *testing.T) {
	src := resources.Secret{
		Scope:       "my_scope",
		Key:         "my_key",
		StringValue: "my_secret_value",
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretConverter{}.Convert(ctx, "my_secret", vin, out)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"scope":        "my_scope",
		"key":          "my_key",
		"string_value": "my_secret_value",
	}, out.Secret["my_secret"])
}

func TestConvertSecretWithLifecycle(t *testing.T) {
	src := resources.Secret{
		Scope:       "my_scope",
		Key:         "my_key",
		StringValue: "my_secret_value",
		BaseResource: resources.BaseResource{
			Lifecycle: resources.Lifecycle{
				PreventDestroy: true,
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = secretConverter{}.Convert(ctx, "my_secret", vin, out)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"scope":        "my_scope",
		"key":          "my_key",
		"string_value": "my_secret_value",
		"lifecycle": map[string]any{
			"prevent_destroy": true,
		},
	}, out.Secret["my_secret"])
}

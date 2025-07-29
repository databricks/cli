package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestRunAsErrorForModelServing(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	ctx := context.Background()
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
	})

	diags := bundle.Apply(ctx, b, resourcemutator.SetRunAs())
	err := diags.Error()

	assert.ErrorContains(t, err, "model_serving_endpoints do not support a setting a run_as user that is different from the owner.\n"+
		"Current identity: jane@doe.com. Run as identity: my_service_principal.\n"+
		"See https://docs")
}

func TestRunAsNoErrorForModelServingEndpoints(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	// We should not error because the model serving endpoint is being deployed
	// with the same identity as the bundle run_as identity.
	ctx := context.Background()
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
	})

	diags := bundle.Apply(ctx, b, resourcemutator.SetRunAs())
	assert.NoError(t, diags.Error())
}

func TestRunAsErrorWhenBothUserAndSpSpecified(t *testing.T) {
	b := load(t, "./run_as/not_allowed/both_sp_and_user")

	ctx := context.Background()
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
	})

	diags := bundle.Apply(ctx, b, resourcemutator.SetRunAs())
	err := diags.Error()

	assert.ErrorContains(t, err, "run_as section cannot specify both user_name and service_principal_name")
}

func TestRunAsErrorNeitherUserOrSpSpecified(t *testing.T) {
	tcases := []struct {
		name string
		err  string
	}{
		{
			name: "empty_run_as",
			err:  "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified",
		},
		{
			name: "empty_sp",
			err:  "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified",
		},
		{
			name: "empty_user",
			err:  "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified",
		},
		{
			name: "empty_user_and_sp",
			err:  "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			bundlePath := "./run_as/not_allowed/neither_sp_nor_user/" + tc.name
			b := load(t, bundlePath)

			ctx := context.Background()
			bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
				b.Config.Workspace.CurrentUser = &config.User{
					User: &iam.User{
						UserName: "my_service_principal",
					},
				}
			})

			diags := bundle.Apply(ctx, b, resourcemutator.SetRunAs())
			err := diags.Error()
			assert.EqualError(t, err, tc.err)
		})
	}
}

func TestRunAsErrorNeitherUserOrSpSpecifiedAtTargetOverride(t *testing.T) {
	b := loadTarget(t, "./run_as/not_allowed/neither_sp_nor_user/override", "development")

	ctx := context.Background()
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
	})

	diags := bundle.Apply(ctx, b, resourcemutator.SetRunAs())
	err := diags.Error()

	assert.EqualError(t, err, "run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified")
}

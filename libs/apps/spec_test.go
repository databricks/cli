package apps

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/require"
)

func TestAppSpecLoadEnvVars(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		AppPath: tempDir,
	}

	tests := []struct {
		name     string
		setup    func(ctx context.Context) (*AppSpec, context.Context, []string)
		wantErr  bool
		checkEnv func(t *testing.T, env []string)
	}{
		{
			name: "direct value environment variables",
			setup: func(ctx context.Context) (*AppSpec, context.Context, []string) {
				spec := &AppSpec{
					config: config,
					EnvVars: []AppEnvVar{
						{
							Name:  "VAR1",
							Value: stringPtr("value1"),
						},
						{
							Name:  "VAR2",
							Value: stringPtr("value2"),
						},
					},
				}
				return spec, ctx, []string{"EXISTING_VAR=existing"}
			},
			wantErr: false,
			checkEnv: func(t *testing.T, env []string) {
				require.Contains(t, env, "VAR1=value1")
				require.Contains(t, env, "VAR2=value2")
				require.Contains(t, env, "EXISTING_VAR=existing")
			},
		},
		{
			name: "valueFrom environment variables",
			setup: func(ctx context.Context) (*AppSpec, context.Context, []string) {
				spec := &AppSpec{
					config: config,
					EnvVars: []AppEnvVar{
						{
							Name:      "VAR1",
							ValueFrom: stringPtr("VAR1"),
						},
						{
							Name:      "VAR2",
							ValueFrom: stringPtr("VAR2"),
						},
					},
				}
				return spec, ctx, []string{"VAR1=value1", "VAR2=value2"}
			},
			wantErr: false,
			checkEnv: func(t *testing.T, env []string) {
				require.Contains(t, env, "VAR1=value1")
				require.Contains(t, env, "VAR2=value2")
			},
		},
		{
			name: "mixed environment variables",
			setup: func(ctx context.Context) (*AppSpec, context.Context, []string) {
				spec := &AppSpec{
					config: config,
					EnvVars: []AppEnvVar{
						{
							Name:  "VAR1",
							Value: stringPtr("value1"),
						},
						{
							Name:      "VAR2",
							ValueFrom: stringPtr("VAR2"),
						},
					},
				}
				return spec, ctx, []string{"VAR2=value2"}
			},
			wantErr: false,
			checkEnv: func(t *testing.T, env []string) {
				require.Contains(t, env, "VAR1=value1")
				require.Contains(t, env, "VAR2=value2")
			},
		},
		{
			name: "environment variables set in env",
			setup: func(ctx context.Context) (*AppSpec, context.Context, []string) {
				spec := &AppSpec{
					config: config,
					EnvVars: []AppEnvVar{
						{
							Name:      "VAR1",
							ValueFrom: stringPtr("VAR1"),
						},
					},
				}
				ctx = env.Set(ctx, "VAR1", "value1")
				return spec, ctx, nil
			},
			wantErr: false,
			checkEnv: func(t *testing.T, env []string) {
				require.Contains(t, env, "VAR1=value1")
			},
		},
		{
			name: "missing valueFrom environment variable",
			setup: func(ctx context.Context) (*AppSpec, context.Context, []string) {
				spec := &AppSpec{
					config: config,
					EnvVars: []AppEnvVar{
						{
							Name:      "VAR1",
							ValueFrom: stringPtr("MISSING_VAR"),
						},
					},
				}
				return spec, ctx, []string{}
			},
			wantErr: true,
			checkEnv: func(t *testing.T, env []string) {
				// Should not reach here as we expect an error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			spec, ctx, customEnv := tt.setup(ctx)
			env, err := spec.LoadEnvVars(ctx, customEnv)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkEnv(t, env)
			}
		})
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}

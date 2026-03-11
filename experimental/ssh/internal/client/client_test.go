package client_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    client.ClientOptions
		wantErr string
	}{
		{
			name:    "no cluster or connection name or accelerator",
			opts:    client.ClientOptions{},
			wantErr: "please provide --cluster or --accelerator flag",
		},
		{
			name: "proxy mode skips cluster/name check",
			opts: client.ClientOptions{ProxyMode: true},
		},
		{
			name: "cluster ID only",
			opts: client.ClientOptions{ClusterID: "abc-123"},
		},
		{
			name:    "accelerator with cluster ID",
			opts:    client.ClientOptions{ClusterID: "abc-123", Accelerator: "GPU_1xA10"},
			wantErr: "--accelerator flag can only be used with serverless compute, not with --cluster",
		},
		{
			name: "accelerator only (auto-generate session name)",
			opts: client.ClientOptions{Accelerator: "GPU_1xA10"},
		},
		{
			name: "connection name without accelerator",
			opts: client.ClientOptions{ConnectionName: "my-conn"},
		},
		{
			name:    "invalid connection name characters",
			opts:    client.ClientOptions{ConnectionName: "my conn!", Accelerator: "GPU_1xA10"},
			wantErr: `connection name "my conn!" must consist of letters, numbers, dashes, and underscores`,
		},
		{
			name:    "connection name with leading dash",
			opts:    client.ClientOptions{ConnectionName: "-my-conn", Accelerator: "GPU_1xA10"},
			wantErr: `connection name "-my-conn" must consist of letters, numbers, dashes, and underscores`,
		},
		{
			name: "valid connection name with accelerator",
			opts: client.ClientOptions{ConnectionName: "my-conn_1", Accelerator: "GPU_1xA10"},
		},
		{
			name: "valid connection name with GPU_8xH100 accelerator",
			opts: client.ClientOptions{ConnectionName: "my-conn_1", Accelerator: "GPU_8xH100"},
		},
		{
			name:    "invalid accelerator value",
			opts:    client.ClientOptions{ConnectionName: "my-conn", Accelerator: "CPU_1x"},
			wantErr: `invalid accelerator value: "CPU_1x", expected "GPU_1xA10" or "GPU_8xH100"`,
		},
		{
			name: "both cluster ID and connection name",
			opts: client.ClientOptions{ClusterID: "abc-123", ConnectionName: "my-conn", Accelerator: "GPU_1xA10"},
		},
		{
			name:    "proxy mode with invalid connection name",
			opts:    client.ClientOptions{ProxyMode: true, ConnectionName: "bad name!", Accelerator: "GPU_1xA10"},
			wantErr: `connection name "bad name!" must consist of letters, numbers, dashes, and underscores`,
		},
		{
			name:    "invalid IDE value",
			opts:    client.ClientOptions{ClusterID: "abc-123", IDE: "vim"},
			wantErr: `invalid IDE value: "vim", expected "vscode" or "cursor"`,
		},
		{
			name: "valid IDE vscode",
			opts: client.ClientOptions{ClusterID: "abc-123", IDE: "vscode"},
		},
		{
			name: "valid IDE cursor",
			opts: client.ClientOptions{ClusterID: "abc-123", IDE: "cursor"},
		},
		{
			name:    "environment version too low",
			opts:    client.ClientOptions{ClusterID: "abc-123", EnvironmentVersion: 3},
			wantErr: "environment version must be >= 4, got 3",
		},
		{
			name: "valid environment version",
			opts: client.ClientOptions{ClusterID: "abc-123", EnvironmentVersion: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestToProxyCommand(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)
	quoted := fmt.Sprintf("%q", exe)

	tests := []struct {
		name string
		opts client.ClientOptions
		want string
	}{
		{
			name: "dedicated cluster",
			opts: client.ClientOptions{ClusterID: "abc-123", ShutdownDelay: 5 * time.Minute},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=5m0s",
		},
		{
			name: "dedicated cluster with auto-start",
			opts: client.ClientOptions{ClusterID: "abc-123", AutoStartCluster: true, ShutdownDelay: 5 * time.Minute},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=true --shutdown-delay=5m0s",
		},
		{
			name: "serverless",
			opts: client.ClientOptions{ConnectionName: "my-conn", ShutdownDelay: 2 * time.Minute},
			want: quoted + " ssh connect --proxy --name=my-conn --shutdown-delay=2m0s",
		},
		{
			name: "serverless with accelerator",
			opts: client.ClientOptions{ConnectionName: "my-conn", Accelerator: "GPU_1xA10", ShutdownDelay: 2 * time.Minute},
			want: quoted + " ssh connect --proxy --name=my-conn --shutdown-delay=2m0s --accelerator=GPU_1xA10",
		},
		{
			name: "with metadata",
			opts: client.ClientOptions{ClusterID: "abc-123", ServerMetadata: "user,2222,abc-123"},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=0s --metadata=user,2222,abc-123",
		},
		{
			name: "with handover timeout",
			opts: client.ClientOptions{ClusterID: "abc-123", HandoverTimeout: 10 * time.Minute},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=0s --handover-timeout=10m0s",
		},
		{
			name: "with profile",
			opts: client.ClientOptions{ClusterID: "abc-123", Profile: "my-profile"},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=0s --profile=my-profile",
		},
		{
			name: "with liteswap",
			opts: client.ClientOptions{ClusterID: "abc-123", Liteswap: "test-env"},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=0s --liteswap=test-env",
		},
		{
			name: "with environment version",
			opts: client.ClientOptions{ClusterID: "abc-123", EnvironmentVersion: 4},
			want: quoted + " ssh connect --proxy --cluster=abc-123 --auto-start-cluster=false --shutdown-delay=0s --environment-version=4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.opts.ToProxyCommand()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

package aircmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	rootcmd "github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListComputeOptions(t *testing.T) {
	var workspaceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != computeOptionsPath {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		assert.Equal(t, http.MethodGet, r.Method)
		workspaceID = r.Header.Get("X-Databricks-Workspace-Id")
		_, _ = w.Write([]byte(`{
  "compute_options": [
    {"hardware_accelerator": "GPU_1xA10", "display_name": "1x A10", "multi_node_supported": true},
    {"hardware_accelerator": "GPU_8xH100", "display_name": "8x H100", "multi_node_supported": true}
  ]
}`))
	}))
	t.Cleanup(srv.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        srv.URL,
		Token:       "token",
		WorkspaceID: "12345",
	})
	require.NoError(t, err)

	options, err := listComputeOptions(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, []computeOption{
		{HardwareAccelerator: "GPU_1xA10"},
		{HardwareAccelerator: "GPU_8xH100"},
	}, options)
	assert.Equal(t, "12345", workspaceID)
}

func TestLocalAcceleratorHelpOptionsIntersectsBackendAndRegistry(t *testing.T) {
	options := localAcceleratorHelpOptions([]computeOption{
		{HardwareAccelerator: "GPU_8xH100"},
		{HardwareAccelerator: "GPU_8xB300"},
		{HardwareAccelerator: "GPU_1xA10"},
	})

	assert.Equal(t, []acceleratorHelpOption{
		{typeName: gpuType1xA10, perNodeGPUs: 1},
		{typeName: gpuType8xH100, perNodeGPUs: 8},
	}, options)
}

func TestLookupComputeOptionsForHelpEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	contents := fmt.Sprintf("[empty]\nhost = %s\ntoken = token\nworkspace_id = 12345\n", srv.URL)
	require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.Flags().StringP("profile", "p", "", "")
	require.NoError(t, cmd.Flags().Set("profile", "empty"))

	_, err := lookupComputeOptionsForHelp(cmd)
	assert.ErrorIs(t, err, errNoComputeOptions)
}

func TestWriteComputeOptionsHelpFallsBack(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantWarning string
	}{
		{"empty response", errNoComputeOptions, "Workspace reported no available accelerator types"},
		{"lookup failure", errors.New("endpoint unavailable"), "Couldn't verify workspace accelerator availability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut strings.Builder
			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())

			writeComputeOptionsHelp(&out, &errOut, cmd, func(*cobra.Command) ([]computeOption, error) {
				return nil, tt.err
			})

			assert.Contains(t, errOut.String(), tt.wantWarning)
			assert.Contains(t, out.String(), "workspace availability not verified")
			for _, typeName := range gpuTypes {
				assert.Contains(t, out.String(), string(typeName))
			}
		})
	}
}

func TestWriteRunConfigFieldHelpOutsideComputeDoesNotLookup(t *testing.T) {
	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.SetOut(&out)

	var calls int
	err := writeRunConfigFieldHelp(cmd, "config.compute.accelerator_type", func(*cobra.Command) ([]computeOption, error) {
		calls++
		return nil, nil
	})
	require.NoError(t, err)
	assert.Zero(t, calls)
	assert.Contains(t, out.String(), "Matched case-sensitively.")
	assert.NotContains(t, out.String(), "Accelerator types available")
}

func TestRunComputeHelpHonorsProfile(t *testing.T) {
	selected, selectedCalls := computeOptionsTestServer(t, "GPU_1xA10")
	other, otherCalls := computeOptionsTestServer(t, "GPU_8xH100")

	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	contents := fmt.Sprintf(`
[selected]
host = %s
token = selected-token
workspace_id = 12345

[other]
host = %s
token = other-token
workspace_id = 67890

[__settings__]
default_profile = other
`, selected.URL, other.URL)
	require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_TOKEN", "")

	var output strings.Builder
	cmd := computeHelpRootCommand(t, &output)
	cmd.SetArgs([]string{"experimental", "air", "run", "-h", "config.compute", "--profile", "selected"})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, output.String(), "Accelerator types available in this workspace:")
	assert.Contains(t, output.String(), "GPU_1xA10")
	assert.NotContains(t, output.String(), "GPU_8xH100")
	assert.Equal(t, int32(1), selectedCalls.Load())
	assert.Zero(t, otherCalls.Load())
}

func computeOptionsTestServer(t *testing.T, accelerator string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != computeOptionsPath {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		calls.Add(1)
		_, _ = fmt.Fprintf(w, `{"compute_options":[{"hardware_accelerator":%q}]}`, accelerator)
	}))
	t.Cleanup(srv.Close)
	return srv, calls
}

func computeHelpRootCommand(t *testing.T, output *strings.Builder) *cobra.Command {
	t.Helper()
	cmd := rootcmd.New(t.Context())
	experimental := &cobra.Command{Use: "experimental"}
	experimental.AddCommand(New())
	cmd.AddCommand(experimental)
	cmd.SetOut(output)
	cmd.SetErr(output)
	return cmd
}

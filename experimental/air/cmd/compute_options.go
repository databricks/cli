package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	computeOptionsPath       = "/api/2.0/ai-compute-manager/compute-options"
	computeHelpLookupTimeout = 5 * time.Second
)

var errNoComputeOptions = errors.New("workspace reported no available accelerator types")

type computeOption struct {
	HardwareAccelerator string `json:"hardware_accelerator"`
}

type computeOptionsResponse struct {
	ComputeOptions []computeOption `json:"compute_options"`
}

type acceleratorHelpOption struct {
	typeName    gpuType
	perNodeGPUs int
}

type computeOptionsLookup func(*cobra.Command) ([]computeOption, error)

func writeRunConfigFieldHelp(cmd *cobra.Command, path string, lookup computeOptionsLookup) error {
	field, err := resolveAndWriteConfigFieldHelp(cmd.OutOrStdout(), path)
	if err != nil {
		return err
	}
	if field.path != configHelpRoot+".compute" {
		return nil
	}

	writeComputeOptionsHelp(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd, lookup)
	return nil
}

func writeComputeOptionsHelp(out, errOut io.Writer, cmd *cobra.Command, lookup computeOptionsLookup) {
	options, err := lookup(cmd)
	if err != nil {
		if errors.Is(err, errNoComputeOptions) {
			fmt.Fprintln(errOut, "Warning: Workspace reported no available accelerator types; showing the built-in list instead.")
		} else {
			fmt.Fprintln(errOut, "Warning: Couldn't verify workspace accelerator availability; showing the built-in list instead.")
		}
		renderAcceleratorHelp(out, "Accelerator types supported by this CLI (workspace availability not verified):", localAcceleratorHelpOptions(nil))
		return
	}

	renderAcceleratorHelp(out, "Accelerator types available in this workspace:", localAcceleratorHelpOptions(options))
}

// Passing nil returns the full local registry for fallback help.
func localAcceleratorHelpOptions(backend []computeOption) []acceleratorHelpOption {
	var available map[string]bool
	if backend != nil {
		available = make(map[string]bool, len(backend))
		for _, option := range backend {
			if option.HardwareAccelerator != "" {
				available[option.HardwareAccelerator] = true
			}
		}
	}

	options := make([]acceleratorHelpOption, 0, len(gpuTypes))
	for _, typeName := range gpuTypes {
		if available != nil && !available[string(typeName)] {
			continue
		}
		perNode, err := gpusPerNode(typeName)
		if err != nil {
			continue
		}
		options = append(options, acceleratorHelpOption{typeName: typeName, perNodeGPUs: perNode})
	}
	return options
}

func renderAcceleratorHelp(w io.Writer, heading string, options []acceleratorHelpOption) {
	fmt.Fprintf(w, "\n%s\n", heading)
	if len(options) == 0 {
		fmt.Fprintln(w, "  No accelerator types supported by this CLI were reported.")
		return
	}
	for _, option := range options {
		unit := "accelerators"
		if option.perNodeGPUs == 1 {
			unit = "accelerator"
		}
		fmt.Fprintf(w, "  %-15s  %d %s per node\n", option.typeName, option.perNodeGPUs, unit)
	}
}

func lookupComputeOptionsForHelp(cmd *cobra.Command) ([]computeOption, error) {
	ctx, cancel := context.WithTimeout(cmd.Context(), computeHelpLookupTimeout)
	defer cancel()

	w, err := computeHelpWorkspaceClient(ctx, cmd)
	if err != nil {
		return nil, err
	}
	options, err := listComputeOptions(ctx, w)
	if err != nil {
		return nil, err
	}
	if len(options) == 0 {
		return nil, errNoComputeOptions
	}
	return options, nil
}

func computeHelpWorkspaceClient(ctx context.Context, cmd *cobra.Command) (*databricks.WorkspaceClient, error) {
	timeoutSeconds := int(computeHelpLookupTimeout / time.Second)
	cfg := &config.Config{
		HTTPTimeoutSeconds:  timeoutSeconds,
		RetryTimeoutSeconds: timeoutSeconds,
	}

	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Value.String() != "" {
		cfg.Profile = profileFlag.Value.String()
		cfg.Loaders = databrickscfg.ProfileAuthLoaders
	} else {
		auth.NormalizeDatabricksConfigFromEnv(ctx, cfg)
		root.ResolveDefaultProfile(ctx, cfg)
	}

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return nil, fmt.Errorf("resolve workspace for compute help: %w", err)
	}
	return w, nil
}

func listComputeOptions(ctx context.Context, w *databricks.WorkspaceClient) ([]computeOption, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("create compute options client: %w", err)
	}

	var response computeOptionsResponse
	err = apiClient.Do(ctx, http.MethodGet, computeOptionsPath, auth.WorkspaceIDHeaders(w.Config), nil, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("list compute options: %w", err)
	}
	return response.ComputeOptions, nil
}

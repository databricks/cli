package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func computeRunnableResourcesMap(b *bundle.Bundle, cliType clis.CLIType) map[string]string {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	for k, ref := range resources.Completions(b, run.IsRunnable) {
		if cliType == clis.DLT && ref.Description.SingularTitle != "Pipeline" {
			continue
		}
		title := fmt.Sprintf("%s: %s", ref.Description.SingularTitle, ref.Resource.GetName())
		inv[title] = k
	}
	return inv
}

func promptRunArgument(ctx context.Context, b *bundle.Bundle, cliType clis.CLIType, runnable map[string]string) (string, error) {
	key, err := cmdio.Select(ctx, runnable, "Resource to run")
	if err != nil {
		return "", err
	}

	return key, nil
}

// resolveRunArgument resolves the resource key to run.
// It returns the remaining arguments to pass to the runner, if applicable.
func resolveRunArgument(ctx context.Context, b *bundle.Bundle, args []string, cliType clis.CLIType) (string, []string, error) {
	// DLT CLI: if there is a single pipeline, just run it without prompting.
	runnableResources := computeRunnableResourcesMap(b, cliType)
	if len(args) == 0 && cliType == clis.DLT && len(runnableResources) == 1 {
		return maps.Values(runnableResources)[0], args, nil
	}

	// If no arguments are specified, prompt the user to select something to run.
	if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
		key, err := promptRunArgument(ctx, b, cliType, runnableResources)
		if err != nil {
			return "", nil, err
		}
		return key, args, nil
	}

	if len(args) < 1 {
		return "", nil, errors.New("expected a KEY of the resource to run")
	}

	return args[0], args[1:], nil
}

func keyToRunner(b *bundle.Bundle, arg string) (run.Runner, error) {
	// Locate the resource to run.
	ref, err := resources.Lookup(b, arg, run.IsRunnable)
	if err != nil {
		return nil, err
	}

	// Convert the resource to a runnable resource.
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func newRunCommand(cliType clis.CLIType) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] KEY",
		Short: "Run a job or pipeline update",
		Long: `Run the job or pipeline identified by KEY.

The KEY is the unique identifier of the resource to run. In addition to
customizing the run using any of the available flags, you can also specify
keyword or positional arguments as shown in these examples:

   databricks bundle run my_job -- --key1 value1 --key2 value2

Or:

   databricks bundle run my_job -- value1 value2 value3

If the specified job uses job parameters or the job has a notebook task with
parameters, the first example applies and flag names are mapped to the
parameter names.

If the specified job does not use job parameters and the job has a Python file
task or a Python wheel task, the second example applies.
`,
	}
	if cliType == clis.DLT {
		cmd.Short = "Run a DLT update"
		cmd.Long = `Run the DLT identified by KEY.

Example: dlt run my_dlt`
	}

	var runOptions run.Options
	runOptions.Define(cmd, cliType)

	var noWait bool
	var restart bool
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Don't wait for the run to complete.")
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart the run if it is already running.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = phases.Initialize(ctx, b)
		if err := diags.Error(); err != nil {
			return err
		}

		key, args, err := resolveRunArgument(ctx, b, args, cliType)
		if err != nil {
			return err
		}

		diags = bundle.ApplySeq(ctx, b,
			terraform.Interpolate(),
			terraform.Write(),
			terraform.StatePull(),
			terraform.Load(terraform.ErrorOnEmptyState),
		)
		if err := diags.Error(); err != nil {
			return err
		}

		runner, err := keyToRunner(b, key)
		if err != nil {
			return err
		}

		// Parse additional positional arguments.
		err = runner.ParseArgs(args, &runOptions)
		if err != nil {
			return err
		}

		if b.Config.DeployOnRun {
			err = deployOnRun(ctx, b, cliType)
			if err != nil {
				return err
			}
		}

		runOptions.NoWait = noWait
		var output output.RunOutput
		if restart {
			output, err = runner.Restart(ctx, &runOptions)
		} else {
			output, err = runner.Run(ctx, &runOptions)
		}
		if err != nil {
			return err
		}

		if output != nil {
			switch root.OutputType(cmd) {
			case flags.OutputText:
				resultString, err := output.String()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte(resultString))
				if err != nil {
					return err
				}
			case flags.OutputJSON:
				b, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(b)
				if err != nil {
					return err
				}
				_, _ = cmd.OutOrStdout().Write([]byte{'\n'})
			default:
				return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
			}
		}
		return nil
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		b, diags := root.MustConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			cobra.CompErrorln(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 0 {
			completions := resources.Completions(b, run.IsRunnable)
			return maps.Keys(completions), cobra.ShellCompDirectiveNoFileComp
		} else {
			// If we know the resource to run, we can complete additional positional arguments.
			runner, err := keyToRunner(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}

func deployOnRun(ctx context.Context, b *bundle.Bundle, cliType clis.CLIType) error {
	changesDetected, err := detectChanges(ctx, b)
	if err != nil {
		return err
	}

	if changesDetected {
		cmdio.LogString(ctx, fmt.Sprintf("Deploying to target '%s' since deploy_on_run is enabled for this project...", b.Config.Bundle.Target))
		diags := phases.Build(ctx, b)
		diags = diags.Extend(phases.Deploy(ctx, b, nil, cliType))
		if diags.HasError() {
			return diags.Error()
		}
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("No changes detected for target '%s', skipping deployment", b.Config.Bundle.Target))
	}
	return nil
}

// detectChanges checks if there are any changes to the files that have not been deployed yet.
// HACK: the logic here is a bit crude; we should refine it to be more accurate.
func detectChanges(ctx context.Context, b *bundle.Bundle) (bool, error) {
	sync, err := files.GetSync(ctx, b)
	if err != nil {
		return false, err
	}

	list, err := sync.GetFileList(ctx)
	if err != nil {
		return false, err
	}

	stateFile, err := deploy.GetPathToStateFile(ctx, b)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(stateFile)
	if err != nil {
		return false, err
	}

	changesDetected := false
	for _, file := range list {
		if file.Modified().After(info.ModTime()) {
			changesDetected = true
			break
		}
	}

	return changesDetected, nil
}

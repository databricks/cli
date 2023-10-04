package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: `The Apps API allows you to deploy, update, and delete lakehouse apps.`,
		Long: `The Apps API allows you to deploy, update, and delete Lakehouse Apps.

  TODO`,
		GroupID: "apps",
		Annotations: map[string]string{
			"package": "apps",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start deploy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deployOverrides []func(
	*cobra.Command,
	*DeployAppRequest,
)

func newDeploy() *cobra.Command {
	cmd := &cobra.Command{}

	var deployReq DeployAppRequest
	var manifestYaml flags.YamlFlag
	var resourcesYaml flags.YamlFlag

	// TODO: short flags
	cmd.Flags().Var(&manifestYaml, "manifest", `path/to/manifest.yaml`)
	cmd.Flags().Var(&resourcesYaml, "resources", `path/to/resources.yaml`)

	cmd.Use = "deploy"
	cmd.Short = `Deploy an app.`
	cmd.Long = `Deploy an Lakehouse App.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("manifest") {
			err = manifestYaml.Unmarshal(&deployReq.Manifest)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in YAML format by specifying the --manifest flag")
		}

		if cmd.Flags().Changed("resources") {
			err = resourcesYaml.Unmarshal(&deployReq.Resources)
			if err != nil {
				return err
			}
		}
		c, err := client.New(w.Config)
		if err != nil {
			return err
		}
		apps := NewApps(c)
		resp, err := apps.Deploy(ctx, &deployReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, resp)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deployOverrides {
		fn(cmd, &deployReq)
	}

	return cmd
}

// start delete command

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq DeleteAppRequest

	// TODO: short flags

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an app.`
	cmd.Long = `Delete an app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		c, err := client.New(w.Config)
		if err != nil {
			return err
		}
		apps := NewApps(c)
		resp, err := apps.Delete(ctx, &deleteReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, resp)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

// start get command

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq GetAppRequest

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get or list app.`
	cmd.Long = `Get or list app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.MaximumNArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) > 0 {
			getReq.Name = args[0]
		}

		c, err := client.New(w.Config)
		if err != nil {
			return err
		}
		apps := NewApps(c)

		resp, err := apps.Get(ctx, &getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, resp)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeploy())
		cmd.AddCommand(newDelete())
		cmd.AddCommand(newGet())
	})
}

// end service ServingEndpoints

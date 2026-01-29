package psql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	lakebasepsql "github.com/databricks/cli/libs/lakebase/psql"
	lakebasev1 "github.com/databricks/cli/libs/lakebase/v1"
	lakebasev2 "github.com/databricks/cli/libs/lakebase/v2"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	return newLakebaseConnectCommand()
}

func newLakebaseConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "psql [TARGET] [-- PSQL_ARGS...]",
		Short:   "Connect to a Database Instance or Postgres endpoint",
		GroupID: "database",
		Long: `Connect to a Database Instance or Postgres endpoint.

This command requires a psql client to be installed on your machine for the connection to work.

The command includes automatic retry logic for connection failures. You can configure the retry behavior using the flags below.

Usage modes:

1. Old Database API (existing behavior):
   databricks psql my-database-instance

2. New Postgres API with full endpoint path:
   databricks psql projects/my-project/branches/main/endpoints/primary

3. New Postgres API using flags (auto-selects default branch and read-write endpoint):
   databricks psql --project my-project
   databricks psql --project my-project --branch main
   databricks psql --project my-project --branch main --endpoint primary

4. Interactive selection (shows dropdown - lists both database instances and projects):
   databricks psql

You can pass additional arguments to psql after a double-dash (--):
  databricks psql my-database -- -c "SELECT * FROM my_table"
  databricks psql --project my-project -- --echo-all -d "my-db"
`,
	}

	// Add retry configuration flag
	cmd.Flags().Int("max-retries", 3, "Maximum number of connection retry attempts (set to 0 to disable retries)")

	// Add Postgres API flags
	cmd.Flags().String("project", "", "Postgres project ID (triggers new API)")
	cmd.Flags().String("branch", "", "Postgres branch ID (default: auto-select default branch)")
	cmd.Flags().String("endpoint", "", "Postgres endpoint ID (default: auto-select read-write endpoint)")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		argsLenAtDash := cmd.ArgsLenAtDash()

		// If -- was used, only count args before the dash
		var argsBeforeDash int
		if argsLenAtDash >= 0 {
			argsBeforeDash = argsLenAtDash
		} else {
			argsBeforeDash = len(args)
		}

		// Read flags
		projectFlag, _ := cmd.Flags().GetString("project")
		branchFlag, _ := cmd.Flags().GetString("branch")
		endpointFlag, _ := cmd.Flags().GetString("endpoint")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")

		retryConfig := lakebasepsql.RetryConfig{
			MaxRetries:    maxRetries,
			InitialDelay:  time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
		}

		// Get extra args for psql
		var extraArgs []string
		if argsBeforeDash < len(args) {
			extraArgs = args[argsBeforeDash:]
		}

		// Determine which API to use based on input
		// 1. If --project, --branch, or --endpoint flags are set -> New Postgres API
		if projectFlag != "" || branchFlag != "" || endpointFlag != "" {
			if projectFlag == "" {
				return errors.New("--project is required when using --branch or --endpoint")
			}
			return connectViaPostgresAPI(ctx, projectFlag, branchFlag, endpointFlag, retryConfig, extraArgs)
		}

		// 2. If positional arg starts with "projects/" -> New Postgres API
		if argsBeforeDash == 1 {
			target := args[0]
			if strings.HasPrefix(target, "projects/") {
				return connectViaPostgresPath(ctx, target, retryConfig, extraArgs)
			}
			// Positional arg is database instance name -> Old Database API
			return connectViaDatabaseAPI(ctx, target, retryConfig, extraArgs)
		}

		// 3. No args, no flags -> Show combined dropdown
		if argsBeforeDash == 0 {
			return showCombinedSelectionAndConnect(ctx, retryConfig, extraArgs)
		}

		return errors.New("expected at most one positional argument for target")
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		err := root.MustWorkspaceClient(cmd, args)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		var names []string

		// Add database instances
		instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
		if err == nil {
			for _, instance := range instances {
				names = append(names, instance.Name)
			}
		}

		// Add postgres projects
		projects, err := w.Postgres.ListProjectsAll(ctx, postgres.ListProjectsRequest{})
		if err == nil {
			for _, project := range projects {
				names = append(names, project.Name)
			}
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

// connectViaDatabaseAPI connects using the old Database API.
func connectViaDatabaseAPI(ctx context.Context, instanceName string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	db, err := lakebasev1.GetDatabaseInstance(ctx, w, instanceName)
	if err != nil {
		return err
	}

	return lakebasev1.Connect(ctx, w, db, retryConfig, extraArgs...)
}

// connectViaPostgresAPI connects using the new Postgres API with flags.
func connectViaPostgresAPI(ctx context.Context, projectID, branchID, endpointID string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	endpoint, err := resolveEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return err
	}

	return lakebasev2.Connect(ctx, w, endpoint, retryConfig, extraArgs...)
}

// connectViaPostgresPath connects using the new Postgres API with a resource path.
func connectViaPostgresPath(ctx context.Context, path string, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	// Parse the resource path to extract project, branch, endpoint IDs
	projectID, branchID, endpointID := parseResourcePath(path)
	if projectID == "" {
		return fmt.Errorf("invalid resource path: %s", path)
	}

	endpoint, err := resolveEndpoint(ctx, w, projectID, branchID, endpointID)
	if err != nil {
		return err
	}

	return lakebasev2.Connect(ctx, w, endpoint, retryConfig, extraArgs...)
}

// resolveEndpoint resolves a partial specification to a full endpoint.
// Uses interactive selection when components are missing.
func resolveEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	projectName := "projects/" + projectID

	// If branch not specified, select one
	if branchID == "" {
		branch, err := selectBranch(ctx, w, projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to select branch: %w", err)
		}
		branchID = lakebasev2.ExtractIDFromName(branch.Name, "branches")
	}

	branchName := fmt.Sprintf("%s/branches/%s", projectName, branchID)

	// If endpoint not specified, select one
	if endpointID == "" {
		endpoint, err := selectEndpoint(ctx, w, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to select endpoint: %w", err)
		}
		return endpoint, nil
	}

	return lakebasev2.GetEndpoint(ctx, w, projectID, branchID, endpointID)
}

// selectBranch auto-selects if there's only one branch, otherwise prompts user to select.
func selectBranch(ctx context.Context, w *databricks.WorkspaceClient, projectName string) (*postgres.Branch, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading branches...")
	branches, err := w.Postgres.ListBranchesAll(ctx, postgres.ListBranchesRequest{
		Parent: projectName,
	})
	sp.Close()
	if err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return nil, errors.New("no branches found in project")
	}

	// Auto-select if there's only one branch
	if len(branches) == 1 {
		branchID := lakebasev2.ExtractIDFromName(branches[0].Name, "branches")
		cmdio.LogString(ctx, "Selected branch: "+branchID)
		return &branches[0], nil
	}

	// Multiple branches, prompt user to select
	var items []cmdio.Tuple
	for _, branch := range branches {
		branchID := lakebasev2.ExtractIDFromName(branch.Name, "branches")
		items = append(items, cmdio.Tuple{Name: branchID, Id: branchID})
	}

	selectedID, err := cmdio.SelectOrdered(ctx, items, "Select branch")
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "Selected branch: "+selectedID)

	for i := range branches {
		if lakebasev2.ExtractIDFromName(branches[i].Name, "branches") == selectedID {
			return &branches[i], nil
		}
	}

	return nil, errors.New("selected branch not found")
}

// selectEndpoint auto-selects if there's only one endpoint, otherwise prompts user to select.
func selectEndpoint(ctx context.Context, w *databricks.WorkspaceClient, branchName string) (*postgres.Endpoint, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading endpoints...")
	endpoints, err := w.Postgres.ListEndpointsAll(ctx, postgres.ListEndpointsRequest{
		Parent: branchName,
	})
	sp.Close()
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, errors.New("no endpoints found in branch")
	}

	// Auto-select if there's only one endpoint
	if len(endpoints) == 1 {
		endpointID := lakebasev2.ExtractIDFromName(endpoints[0].Name, "endpoints")
		cmdio.LogString(ctx, "Selected endpoint: "+endpointID)
		return &endpoints[0], nil
	}

	// Multiple endpoints, prompt user to select
	var items []cmdio.Tuple
	for _, endpoint := range endpoints {
		endpointID := lakebasev2.ExtractIDFromName(endpoint.Name, "endpoints")
		items = append(items, cmdio.Tuple{Name: endpointID, Id: endpointID})
	}

	selectedID, err := cmdio.SelectOrdered(ctx, items, "Select endpoint")
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "Selected endpoint: "+selectedID)

	for i := range endpoints {
		if lakebasev2.ExtractIDFromName(endpoints[i].Name, "endpoints") == selectedID {
			return &endpoints[i], nil
		}
	}

	return nil, errors.New("selected endpoint not found")
}

// parseResourcePath extracts project, branch, and endpoint IDs from a resource path.
func parseResourcePath(input string) (project, branch, endpoint string) {
	parts := strings.Split(input, "/")

	// projects/{project_id}
	if len(parts) >= 2 && parts[0] == "projects" {
		project = parts[1]
	}

	// projects/{project_id}/branches/{branch_id}
	if len(parts) >= 4 && parts[2] == "branches" {
		branch = parts[3]
	}

	// projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}
	if len(parts) >= 6 && parts[4] == "endpoints" {
		endpoint = parts[5]
	}

	return project, branch, endpoint
}

// showCombinedSelectionAndConnect shows a combined dropdown of database instances and postgres projects.
func showCombinedSelectionAndConnect(ctx context.Context, retryConfig lakebasepsql.RetryConfig, extraArgs []string) error {
	w := cmdctx.WorkspaceClient(ctx)

	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading database instances and Postgres projects...")

	// Fetch both in parallel
	type instancesResult struct {
		instances []database.DatabaseInstance
		err       error
	}
	type projectsResult struct {
		projects []postgres.Project
		err      error
	}

	instancesCh := make(chan instancesResult, 1)
	projectsCh := make(chan projectsResult, 1)

	go func() {
		instances, err := w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
		instancesCh <- instancesResult{instances, err}
	}()

	go func() {
		projects, err := w.Postgres.ListProjectsAll(ctx, postgres.ListProjectsRequest{})
		projectsCh <- projectsResult{projects, err}
	}()

	instResult := <-instancesCh
	projResult := <-projectsCh
	sp.Close()

	// Build maps from IDs to full objects
	instancesByID := make(map[string]database.DatabaseInstance)
	projectsByID := make(map[string]postgres.Project)

	// Build ordered selection list
	var items []cmdio.Tuple

	if instResult.err == nil {
		for _, inst := range instResult.instances {
			id := "provisioned:" + inst.Name
			instancesByID[id] = inst
			label := inst.Name + " (provisioned)"
			items = append(items, cmdio.Tuple{Name: label, Id: id})
		}
	}

	if projResult.err == nil {
		for _, proj := range projResult.projects {
			// Extract project ID from name like "projects/my-project"
			projectID := proj.Name
			if parts := strings.Split(proj.Name, "/"); len(parts) >= 2 {
				projectID = parts[1]
			}
			id := "autoscaling:projects/" + projectID
			projectsByID[id] = proj
			// Use display name from API if available, otherwise use project ID
			displayName := projectID
			if proj.Status != nil && proj.Status.DisplayName != "" {
				displayName = proj.Status.DisplayName
			}
			label := displayName + " (autoscaling)"
			items = append(items, cmdio.Tuple{Name: label, Id: id})
		}
	}

	if len(items) == 0 {
		// Build error message
		var errMsgs []string
		if instResult.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to load database instances: %v", instResult.err))
		}
		if projResult.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to load Postgres projects: %v", projResult.err))
		}
		if len(errMsgs) > 0 {
			return fmt.Errorf("could not find any databases: %s", strings.Join(errMsgs, "; "))
		}
		return errors.New("could not find any Database instances or Postgres projects in the workspace")
	}

	selected, err := cmdio.SelectOrdered(ctx, items, "Select database to connect to")
	if err != nil {
		return err
	}

	if inst, ok := instancesByID[selected]; ok {
		cmdio.LogString(ctx, "Selected provisioned database instance: "+inst.Name)
		return lakebasev1.Connect(ctx, w, &inst, retryConfig, extraArgs...)
	}

	if proj, ok := projectsByID[selected]; ok {
		// Extract project ID from name
		projectID := proj.Name
		if parts := strings.Split(proj.Name, "/"); len(parts) >= 2 {
			projectID = parts[1]
		}
		// Use display name for logging
		displayName := projectID
		if proj.Status != nil && proj.Status.DisplayName != "" {
			displayName = proj.Status.DisplayName
		}
		cmdio.LogString(ctx, "Selected autoscaling project: "+displayName)
		return connectViaPostgresAPI(ctx, projectID, "", "", retryConfig, extraArgs)
	}

	return fmt.Errorf("unexpected selection: %s", selected)
}

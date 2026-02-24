package prompt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// DefaultAppDescription is the default description for new apps.
const DefaultAppDescription = "A Databricks App powered by AppKit"

// AppkitTheme returns a custom theme for appkit prompts.
func AppkitTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Databricks brand colors
	red := lipgloss.Color("#BD2B26")
	gray := lipgloss.Color("#71717A") // Mid-tone gray, readable on light and dark
	yellow := lipgloss.Color("#FFAB00")

	t.Focused.Title = t.Focused.Title.Foreground(red).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(gray)

	return t
}

// Styles for printing answered prompts.
var (
	answeredTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#71717A"))
	answeredValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFAB00")).
				Bold(true)
)

// PrintAnswered prints a completed prompt answer to keep history visible.
func PrintAnswered(ctx context.Context, title, value string) {
	cmdio.LogString(ctx, fmt.Sprintf("%s %s", answeredTitleStyle.Render(title+":"), answeredValueStyle.Render(value)))
}

// printAnswered is an alias for internal use.
func printAnswered(ctx context.Context, title, value string) {
	PrintAnswered(ctx, title, value)
}

// RunMode specifies how to run the app after creation.
type RunMode string

const (
	RunModeNone      RunMode = "none"
	RunModeDev       RunMode = "dev"
	RunModeDevRemote RunMode = "dev-remote"
)

// CreateProjectConfig holds the configuration gathered from the interactive prompt.
type CreateProjectConfig struct {
	ProjectName  string
	Description  string
	Features     []string
	Dependencies map[string]string // e.g., {"sql_warehouse_id": "abc123"}
	Deploy       bool              // Whether to deploy the app after creation
	RunMode      RunMode           // How to run the app after creation
}

// App name constraints.
const (
	MaxAppNameLength = 30
	DevTargetPrefix  = "dev-"
)

// projectNamePattern is the compiled regex for validating project names.
// Pre-compiled for efficiency since validation is called on every keystroke.
var projectNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateProjectName validates the project name for length and pattern constraints.
// It checks that the name plus the "dev-" prefix doesn't exceed 30 characters,
// and that the name follows the pattern: starts with a letter, contains only
// lowercase letters, numbers, or hyphens.
func ValidateProjectName(s string) error {
	if s == "" {
		return errors.New("project name is required")
	}

	// Check length constraint (dev- prefix + name <= 30)
	totalLength := len(DevTargetPrefix) + len(s)
	if totalLength > MaxAppNameLength {
		maxAllowed := MaxAppNameLength - len(DevTargetPrefix)
		return fmt.Errorf("name too long (max %d chars)", maxAllowed)
	}

	// Check pattern
	if !projectNamePattern.MatchString(s) {
		return errors.New("must start with a letter, use only lowercase letters, numbers, or hyphens")
	}

	return nil
}

// PrintHeader prints the AppKit header banner.
func PrintHeader(ctx context.Context) {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BD2B26")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A"))

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, headerStyle.Render("◆ Create a new Databricks AppKit project"))
	cmdio.LogString(ctx, subtitleStyle.Render("  Full-stack TypeScript • React • Tailwind CSS"))
	cmdio.LogString(ctx, "")
}

// PromptForProjectName prompts only for project name.
// Used as the first step before resolving templates.
// outputDir is used to check if the destination directory already exists.
func PromptForProjectName(ctx context.Context, outputDir string) (string, error) {
	PrintHeader(ctx)
	theme := AppkitTheme()

	var name string
	err := huh.NewInput().
		Title("App name").
		Description("lowercase letters, numbers, hyphens (max 26 chars)").
		Placeholder("my-app").
		Value(&name).
		Validate(func(s string) error {
			if err := ValidateProjectName(s); err != nil {
				return err
			}
			destDir := s
			if outputDir != "" {
				destDir = filepath.Join(outputDir, s)
			}
			if _, err := os.Stat(destDir); err == nil {
				return fmt.Errorf("directory %s already exists", destDir)
			}
			return nil
		}).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "Project name", name)
	return name, nil
}

// PromptForDeployAndRun prompts for post-creation deploy and run options.
func PromptForDeployAndRun(ctx context.Context) (deploy bool, runMode RunMode, err error) {
	theme := AppkitTheme()

	// Deploy after creation?
	err = huh.NewConfirm().
		Title("Deploy after creation?").
		Description("Run 'databricks apps deploy' after setup").
		Value(&deploy).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, RunModeNone, err
	}
	if deploy {
		printAnswered(ctx, "Deploy after creation", "Yes")
	} else {
		printAnswered(ctx, "Deploy after creation", "No")
	}

	// Build run options - dev-remote requires deploy (needs a deployed app to connect to)
	runOptions := []huh.Option[string]{
		huh.NewOption("No, I'll run it later", string(RunModeNone)),
		huh.NewOption("Yes, run locally (npm run dev)", string(RunModeDev)),
	}
	if deploy {
		runOptions = append(runOptions, huh.NewOption("Yes, run with remote bridge (dev-remote)", string(RunModeDevRemote)))
	}

	// Run the app?
	runModeStr := string(RunModeNone)
	err = huh.NewSelect[string]().
		Title("Run the app after creation?").
		Description("Choose how to start the development server").
		Options(runOptions...).
		Value(&runModeStr).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, RunModeNone, err
	}

	runModeLabels := map[string]string{
		string(RunModeNone):      "No",
		string(RunModeDev):       "Yes (local)",
		string(RunModeDevRemote): "Yes (remote)",
	}
	printAnswered(ctx, "Run after creation", runModeLabels[runModeStr])

	return deploy, RunMode(runModeStr), nil
}

// ListSQLWarehouses fetches all SQL warehouses the user has access to.
func ListSQLWarehouses(ctx context.Context) ([]sql.EndpointInfo, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}

	iter := w.Warehouses.List(ctx, sql.ListWarehousesRequest{})
	return listing.ToSlice(ctx, iter)
}

// PromptFromList shows a picker for items and returns the selected ID.
// If required is false and items are empty, returns ("", nil). If required is true and items are empty, returns an error.
func PromptFromList(ctx context.Context, title, emptyMessage string, items []ListItem, required bool) (string, error) {
	id, _, err := promptFromListWithLabel(ctx, title, emptyMessage, items, required)
	return id, err
}

// promptFromListWithLabel shows a picker and returns both the selected ID and its display label.
func promptFromListWithLabel(ctx context.Context, title, emptyMessage string, items []ListItem, required bool) (string, string, error) {
	if len(items) == 0 {
		if required {
			return "", "", errors.New(emptyMessage)
		}
		return "", "", nil
	}
	theme := AppkitTheme()
	options := make([]huh.Option[string], 0, len(items))
	labels := make(map[string]string)
	for _, it := range items {
		options = append(options, huh.NewOption(it.Label, it.ID))
		labels[it.ID] = it.Label
	}
	var selected string
	err := huh.NewSelect[string]().
		Title(title).
		Description(fmt.Sprintf("%d available — type to filter", len(items))).
		Options(options...).
		Value(&selected).
		Filtering(true).
		Height(8).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", "", err
	}
	printAnswered(ctx, title, labels[selected])
	return selected, labels[selected], nil
}

// PromptForWarehouse shows a picker to select a SQL warehouse.
func PromptForWarehouse(ctx context.Context) (string, error) {
	var items []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching SQL warehouses...", func() error {
		var fetchErr error
		items, fetchErr = ListSQLWarehousesItems(ctx)
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch SQL warehouses: %w", err)
	}
	return PromptFromList(ctx, "Select SQL Warehouse", "no SQL warehouses found. Create one in your workspace first", items, true)
}

// singleValueResult wraps a single value into the resource values map.
// Uses the first field name from Fields for the composite key (resource_key.field),
// or falls back to the resource key if no Fields are defined.
func singleValueResult(r manifest.Resource, value string) map[string]string {
	if value == "" {
		return nil
	}
	names := r.FieldNames()
	if len(names) >= 1 {
		return map[string]string{r.Key() + "." + names[0]: value}
	}
	return map[string]string{r.Key(): value}
}

// promptForResourceFromLister runs a spinner, fetches items via fn, then shows PromptFromList.
func promptForResourceFromLister(ctx context.Context, r manifest.Resource, required bool, title, emptyMsg, spinnerMsg string, fn func(context.Context) ([]ListItem, error)) (map[string]string, error) {
	var items []ListItem
	err := RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
		var fetchErr error
		items, fetchErr = fn(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	value, err := PromptFromList(ctx, title, emptyMsg, items, required)
	if err != nil {
		return nil, err
	}
	return singleValueResult(r, value), nil
}

// PromptForSecret shows a two-step picker for secret scope and key.
func PromptForSecret(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick scope
	var scopes []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching secret scopes...", func() error {
		var fetchErr error
		scopes, fetchErr = ListSecretScopes(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	scope, err := PromptFromList(ctx, "Select Secret Scope", "no secret scopes found", scopes, required)
	if err != nil {
		return nil, err
	}
	if scope == "" {
		return nil, nil
	}

	// Step 2: pick key within scope
	var keys []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching secret keys...", func() error {
		var fetchErr error
		keys, fetchErr = ListSecretKeys(ctx, scope)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	key, err := PromptFromList(ctx, "Select Secret Key", "no keys found in scope "+scope, keys, required)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}

	return map[string]string{
		r.Key() + ".scope": scope,
		r.Key() + ".key":   key,
	}, nil
}

// PromptForJob shows a picker for jobs.
func PromptForJob(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Job", "no jobs found", "Fetching jobs...", ListJobs)
}

// PromptForSQLWarehouseResource shows a picker for SQL warehouses (manifest.Resource version).
func PromptForSQLWarehouseResource(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select SQL Warehouse", "no SQL warehouses found. Create one in your workspace first", "Fetching SQL warehouses...", ListSQLWarehousesItems)
}

const backID = "__back__"

// promptUCCatalog shows a picker for UC catalogs (shared first step for volume/function pickers).
func promptUCCatalog(ctx context.Context, required bool) (string, error) {
	var items []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching catalogs...", func() error {
		var fetchErr error
		items, fetchErr = ListCatalogs(ctx)
		return fetchErr
	})
	if err != nil {
		return "", err
	}
	return PromptFromList(ctx, "Select Catalog", "no catalogs found", items, required)
}

// promptFromListWithBack shows a picker with a "← Go back" option prepended.
// Returns ("", nil) if the user selects "Go back".
func promptFromListWithBack(ctx context.Context, title string, items []ListItem) (string, error) {
	withBack := make([]ListItem, 0, len(items)+1)
	withBack = append(withBack, ListItem{ID: backID, Label: "← Go back"})
	withBack = append(withBack, items...)
	value, err := PromptFromList(ctx, title, "", withBack, true)
	if err != nil {
		return "", err
	}
	if value == backID {
		return "", nil
	}
	return value, nil
}

// promptUCResource runs a three-step catalog → schema → resource picker with back navigation.
// Empty results at any level show a message and navigate back automatically.
func promptUCResource(ctx context.Context, r manifest.Resource, required bool, resourceLabel, spinnerMsg string, listFn func(context.Context, string, string) ([]ListItem, error)) (map[string]string, error) {
	for {
		catalogName, err := promptUCCatalog(ctx, required)
		if err != nil || catalogName == "" {
			return nil, err
		}

		for {
			var schemas []ListItem
			err = RunWithSpinnerCtx(ctx, "Fetching schemas...", func() error {
				var fetchErr error
				schemas, fetchErr = ListSchemas(ctx, catalogName)
				return fetchErr
			})
			if err != nil {
				return nil, err
			}
			if len(schemas) == 0 {
				cmdio.LogString(ctx, fmt.Sprintf("No schemas found in %s, try another catalog.", catalogName))
				break // back to catalog picker
			}

			schemaName, err := promptFromListWithBack(ctx, "Select Schema", schemas)
			if err != nil {
				return nil, err
			}
			if schemaName == "" {
				break // back to catalog picker
			}

			var items []ListItem
			err = RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
				var fetchErr error
				items, fetchErr = listFn(ctx, catalogName, schemaName)
				return fetchErr
			})
			if err != nil {
				return nil, err
			}
			if len(items) == 0 {
				cmdio.LogString(ctx, fmt.Sprintf("No %ss found in %s.%s, try another schema.", resourceLabel, catalogName, schemaName))
				continue // back to schema picker
			}

			value, err := promptFromListWithBack(ctx, "Select "+resourceLabel, items)
			if err != nil {
				return nil, err
			}
			if value == "" {
				continue // back to schema picker
			}
			return singleValueResult(r, value), nil
		}
	}
}

// PromptForServingEndpoint shows a picker for serving endpoints.
func PromptForServingEndpoint(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Serving Endpoint", "no serving endpoints found", "Fetching serving endpoints...", ListServingEndpoints)
}

// PromptForVolume shows a three-step picker for UC volumes: catalog -> schema -> volume.
func PromptForVolume(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptUCResource(ctx, r, required, "Volume", "Fetching volumes...", ListVolumesInSchema)
}

// PromptForVectorSearchIndex shows a picker for vector search indexes.
func PromptForVectorSearchIndex(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Vector Search Index", "no vector search indexes found", "Fetching vector search indexes...", ListVectorSearchIndexes)
}

// PromptForUCFunction shows a three-step picker for UC functions: catalog -> schema -> function.
func PromptForUCFunction(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptUCResource(ctx, r, required, "UC Function", "Fetching functions...", ListFunctionsInSchema)
}

// PromptForUCConnection shows a picker for UC connections.
func PromptForUCConnection(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select UC Connection", "no connections found", "Fetching connections...", ListConnections)
}

// PromptForDatabase shows a two-step picker for database instance and database name.
func PromptForDatabase(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick a Lakebase instance
	var instances []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching database instances...", func() error {
		var fetchErr error
		instances, fetchErr = ListDatabaseInstances(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	instanceName, err := PromptFromList(ctx, "Select Database Instance", "no database instances found", instances, required)
	if err != nil {
		return nil, err
	}
	if instanceName == "" {
		return nil, nil
	}

	// Step 2: pick a database within the instance
	var databases []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching databases...", func() error {
		var fetchErr error
		databases, fetchErr = ListDatabases(ctx, instanceName)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	dbName, err := PromptFromList(ctx, "Select Database", "no databases found in instance "+instanceName, databases, required)
	if err != nil {
		return nil, err
	}
	if dbName == "" {
		return nil, nil
	}

	return map[string]string{
		r.Key() + ".instance_name": instanceName,
		r.Key() + ".database_name": dbName,
	}, nil
}

// PromptForPostgres shows a three-step picker for Lakebase Autoscaling (V2): project, branch, then database.
func PromptForPostgres(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	// Step 1: pick a project
	var projects []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching Postgres projects...", func() error {
		var fetchErr error
		projects, fetchErr = ListPostgresProjects(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	projectName, err := PromptFromList(ctx, "Select Postgres Project", "no Postgres projects found", projects, required)
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		return nil, nil
	}

	// Step 2: pick a branch within the project
	var branches []ListItem
	err = RunWithSpinnerCtx(ctx, "Fetching branches...", func() error {
		var fetchErr error
		branches, fetchErr = ListPostgresBranches(ctx, projectName)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	branchName, err := PromptFromList(ctx, "Select Branch", "no branches found in project "+projectName, branches, required)
	if err != nil {
		return nil, err
	}
	if branchName == "" {
		return nil, nil
	}

	// Step 3: enter a database name (pre-filled with default)
	dbName := "databricks_postgres"
	theme := AppkitTheme()
	err = huh.NewInput().
		Title("Database name").
		Description("Enter the database name to connect to").
		Value(&dbName).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	if dbName == "" {
		return nil, nil
	}
	printAnswered(ctx, "Database", dbName)

	return map[string]string{
		r.Key() + ".branch":   branchName,
		r.Key() + ".database": dbName,
	}, nil
}

// PromptForGenieSpace shows a picker for Genie spaces.
// Captures both the space ID and name since the DABs schema requires both fields.
func PromptForGenieSpace(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	var items []ListItem
	err := RunWithSpinnerCtx(ctx, "Fetching Genie spaces...", func() error {
		var fetchErr error
		items, fetchErr = ListGenieSpaces(ctx)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}
	id, name, err := promptFromListWithLabel(ctx, "Select Genie Space", "no Genie spaces found", items, required)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return map[string]string{
		r.Key() + ".id":   id,
		r.Key() + ".name": name,
	}, nil
}

// PromptForExperiment shows a picker for MLflow experiments.
func PromptForExperiment(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Experiment", "no experiments found", "Fetching experiments...", ListExperiments)
}

// TODO: uncomment when bundles support app as an app resource type.
// // PromptForAppResource shows a picker for apps (manifest.Resource version).
// func PromptForAppResource(ctx context.Context, r manifest.Resource, required bool) (map[string]string, error) {
// 	return promptForResourceFromLister(ctx, r, required, "Select App", "no apps found. Create one first with 'databricks apps create <name>'", "Fetching apps...", ListAppsItems)
// }

// RunWithSpinnerCtx runs a function while showing a spinner with the given title.
// The spinner stops and the function returns early if the context is cancelled.
// Panics in the action are recovered and returned as errors.
func RunWithSpinnerCtx(ctx context.Context, title string, action func() error) error {
	spinner := cmdio.NewSpinner(ctx)
	spinner.Update(title)

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("action panicked: %v", r)
			}
		}()
		done <- action()
	}()

	select {
	case err := <-done:
		spinner.Close()
		return err
	case <-ctx.Done():
		spinner.Close()
		// Wait for action goroutine to complete to avoid orphaned goroutines.
		// For exec.CommandContext, the process is killed when context is cancelled.
		<-done
		return ctx.Err()
	}
}

// ListAllApps fetches all apps the user has access to from the workspace.
func ListAllApps(ctx context.Context) ([]apps.App, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}

	iter := w.Apps.List(ctx, apps.ListAppsRequest{})
	return listing.ToSlice(ctx, iter)
}

// PromptForAppSelection shows a picker to select an existing app.
// Returns the selected app name or error if cancelled/no apps found.
func PromptForAppSelection(ctx context.Context, title string) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("--name is required in non-interactive mode")
	}

	// Fetch all apps the user has access to
	var existingApps []apps.App
	err := RunWithSpinnerCtx(ctx, "Fetching apps...", func() error {
		var fetchErr error
		existingApps, fetchErr = ListAllApps(ctx)
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch apps: %w", err)
	}

	if len(existingApps) == 0 {
		return "", errors.New("no apps found. Create one first with 'databricks apps create <name>'")
	}

	theme := AppkitTheme()

	// Build options
	options := make([]huh.Option[string], 0, len(existingApps))
	for _, app := range existingApps {
		label := app.Name
		if app.Description != "" {
			desc := app.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			label += " — " + desc
		}
		options = append(options, huh.NewOption(label, app.Name))
	}

	var selected string
	err = huh.NewSelect[string]().
		Title(title).
		Description(fmt.Sprintf("%d apps found — type to filter", len(existingApps))).
		Options(options...).
		Value(&selected).
		Filtering(true).
		Height(8).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "App", selected)
	return selected, nil
}

// PrintSuccess prints a success message after project creation.
// If nextStepsCmd is non-empty, also prints the "Next steps" section with the given command.
func PrintSuccess(ctx context.Context, projectName, outputDir string, fileCount int, nextStepsCmd string) {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFAB00")). // Databricks yellow
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A")) // Mid-tone gray

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF3621")) // Databricks orange

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, successStyle.Render("✔ Project created successfully!"))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, dimStyle.Render("  Location: "+outputDir))
	cmdio.LogString(ctx, dimStyle.Render("  Files: "+strconv.Itoa(fileCount)))

	if nextStepsCmd != "" {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, dimStyle.Render("  Next steps:"))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, codeStyle.Render("    cd "+projectName))
		cmdio.LogString(ctx, codeStyle.Render("    "+nextStepsCmd))
	}
	cmdio.LogString(ctx, "")
}

// SetupNote holds the display name and message for a single plugin setup note.
type SetupNote struct {
	Name    string
	Message string
}

// PrintSetupNotes renders a styled "Setup Notes" section for selected plugins.
func PrintSetupNotes(ctx context.Context, notes []SetupNote) {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFAB00")). // Databricks yellow
		Bold(true)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A")). // Mid-tone gray
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A")) // Mid-tone gray

	cmdio.LogString(ctx, headerStyle.Render("  Setup Notes"))
	cmdio.LogString(ctx, "")
	for _, n := range notes {
		cmdio.LogString(ctx, nameStyle.Render("  "+n.Name))
		indented := strings.ReplaceAll(n.Message, "\n", "\n  ")
		cmdio.LogString(ctx, msgStyle.Render("  "+indented))
		cmdio.LogString(ctx, "")
	}
}

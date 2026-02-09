package prompt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

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
	if len(items) == 0 {
		if required {
			return "", errors.New(emptyMessage)
		}
		return "", nil
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
		return "", err
	}
	printAnswered(ctx, title, labels[selected])
	return selected, nil
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

// promptForResourceFromLister runs a spinner, fetches items via fn, then shows PromptFromList.
func promptForResourceFromLister(ctx context.Context, _ manifest.Resource, required bool, title, emptyMsg, spinnerMsg string, fn func(context.Context) ([]ListItem, error)) (string, error) {
	var items []ListItem
	err := RunWithSpinnerCtx(ctx, spinnerMsg, func() error {
		var fetchErr error
		items, fetchErr = fn(ctx)
		return fetchErr
	})
	if err != nil {
		return "", err
	}
	return PromptFromList(ctx, title, emptyMsg, items, required)
}

// PromptForSecret shows a picker for secret scopes.
func PromptForSecret(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Secret Scope", "no secret scopes found", "Fetching secret scopes...", ListSecrets)
}

// PromptForJob shows a picker for jobs.
func PromptForJob(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Job", "no jobs found", "Fetching jobs...", ListJobs)
}

// PromptForSQLWarehouseResource shows a picker for SQL warehouses (manifest.Resource version).
func PromptForSQLWarehouseResource(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select SQL Warehouse", "no SQL warehouses found. Create one in your workspace first", "Fetching SQL warehouses...", ListSQLWarehousesItems)
}

// PromptForServingEndpoint shows a picker for serving endpoints.
func PromptForServingEndpoint(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Serving Endpoint", "no serving endpoints found", "Fetching serving endpoints...", ListServingEndpoints)
}

// PromptForVolume shows a picker for UC volumes.
func PromptForVolume(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Volume", "no volumes found", "Fetching volumes...", ListVolumes)
}

// PromptForVectorSearchIndex shows a picker for vector search indexes.
func PromptForVectorSearchIndex(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Vector Search Index", "no vector search indexes found", "Fetching vector search indexes...", ListVectorSearchIndexes)
}

// PromptForUCFunction shows a picker for UC functions.
func PromptForUCFunction(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select UC Function", "no functions found", "Fetching functions...", ListFunctions)
}

// PromptForUCConnection shows a picker for UC connections.
func PromptForUCConnection(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select UC Connection", "no connections found", "Fetching connections...", ListConnections)
}

// PromptForDatabase shows a picker for UC catalogs (databases).
func PromptForDatabase(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Database (Catalog)", "no catalogs found", "Fetching catalogs...", ListDatabases)
}

// PromptForGenieSpace shows a picker for Genie spaces.
func PromptForGenieSpace(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Genie Space", "no Genie spaces found", "Fetching Genie spaces...", ListGenieSpaces)
}

// PromptForExperiment shows a picker for MLflow experiments.
func PromptForExperiment(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select Experiment", "no experiments found", "Fetching experiments...", ListExperiments)
}

// PromptForAppResource shows a picker for apps (manifest.Resource version).
func PromptForAppResource(ctx context.Context, r manifest.Resource, required bool) (string, error) {
	return promptForResourceFromLister(ctx, r, required, "Select App", "no apps found. Create one first with 'databricks apps create <name>'", "Fetching apps...", ListAppsItems)
}

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

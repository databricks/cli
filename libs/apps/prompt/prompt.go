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
	"github.com/databricks/cli/libs/apps/features"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
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

// PromptForPluginDependencies prompts for dependencies required by detected plugins.
// Returns a map of dependency ID to value.
func PromptForPluginDependencies(ctx context.Context, deps []features.FeatureDependency) (map[string]string, error) {
	theme := AppkitTheme()
	result := make(map[string]string)

	for _, dep := range deps {
		// Special handling for SQL warehouse - show picker instead of text input
		if dep.ID == "sql_warehouse_id" {
			warehouseID, err := PromptForWarehouse(ctx)
			if err != nil {
				return nil, err
			}
			result[dep.ID] = warehouseID
			continue
		}

		var value string
		description := dep.Description
		if !dep.Required {
			description += " (optional)"
		}

		input := huh.NewInput().
			Title(dep.Title).
			Description(description).
			Placeholder(dep.Placeholder).
			Value(&value)

		if dep.Required {
			input = input.Validate(func(s string) error {
				if s == "" {
					return errors.New("this field is required")
				}
				return nil
			})
		}

		if err := input.WithTheme(theme).Run(); err != nil {
			return nil, err
		}
		printAnswered(ctx, dep.Title, value)
		result[dep.ID] = value
	}

	return result, nil
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

// PromptForProjectConfig shows an interactive form to gather project configuration.
// Flow: name -> features -> feature dependencies -> description -> deploy/run.
// If preSelectedFeatures is provided, the feature selection prompt is skipped.
func PromptForProjectConfig(ctx context.Context, preSelectedFeatures []string) (*CreateProjectConfig, error) {
	config := &CreateProjectConfig{
		Dependencies: make(map[string]string),
		Features:     preSelectedFeatures,
	}
	theme := AppkitTheme()

	PrintHeader(ctx)

	// Step 1: Project name
	err := huh.NewInput().
		Title("Project name").
		Description("lowercase letters, numbers, hyphens (max 26 chars)").
		Placeholder("my-app").
		Value(&config.ProjectName).
		Validate(ValidateProjectName).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	printAnswered(ctx, "Project name", config.ProjectName)

	// Step 2: Feature selection (skip if features already provided via flag)
	if len(config.Features) == 0 && len(features.AvailableFeatures) > 0 {
		options := make([]huh.Option[string], 0, len(features.AvailableFeatures))
		for _, f := range features.AvailableFeatures {
			label := f.Name + " - " + f.Description
			options = append(options, huh.NewOption(label, f.ID))
		}

		err = huh.NewMultiSelect[string]().
			Title("Select features").
			Description("space to toggle, enter to confirm").
			Options(options...).
			Value(&config.Features).
			Height(8).
			WithTheme(theme).
			Run()
		if err != nil {
			return nil, err
		}
		if len(config.Features) == 0 {
			printAnswered(ctx, "Features", "None")
		} else {
			printAnswered(ctx, "Features", fmt.Sprintf("%d selected", len(config.Features)))
		}
	}

	// Step 3: Prompt for feature dependencies
	deps := features.CollectDependencies(config.Features)
	for _, dep := range deps {
		// Special handling for SQL warehouse - show picker instead of text input
		if dep.ID == "sql_warehouse_id" {
			warehouseID, err := PromptForWarehouse(ctx)
			if err != nil {
				return nil, err
			}
			config.Dependencies[dep.ID] = warehouseID
			continue
		}

		var value string
		description := dep.Description
		if !dep.Required {
			description += " (optional)"
		}

		input := huh.NewInput().
			Title(dep.Title).
			Description(description).
			Placeholder(dep.Placeholder).
			Value(&value)

		if dep.Required {
			input = input.Validate(func(s string) error {
				if s == "" {
					return errors.New("this field is required")
				}
				return nil
			})
		}

		if err := input.WithTheme(theme).Run(); err != nil {
			return nil, err
		}
		printAnswered(ctx, dep.Title, value)
		config.Dependencies[dep.ID] = value
	}

	// Step 4: Description
	config.Description = DefaultAppDescription
	err = huh.NewInput().
		Title("Description").
		Placeholder(DefaultAppDescription).
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	if config.Description == "" {
		config.Description = DefaultAppDescription
	}
	printAnswered(ctx, "Description", config.Description)

	// Step 5: Deploy after creation?
	err = huh.NewConfirm().
		Title("Deploy after creation?").
		Description("Run 'databricks apps deploy' after setup").
		Value(&config.Deploy).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	if config.Deploy {
		printAnswered(ctx, "Deploy after creation", "Yes")
	} else {
		printAnswered(ctx, "Deploy after creation", "No")
	}

	// Step 6: Run the app?
	runModeStr := string(RunModeNone)
	err = huh.NewSelect[string]().
		Title("Run the app after creation?").
		Description("Choose how to start the development server").
		Options(
			huh.NewOption("No, I'll run it later", string(RunModeNone)),
			huh.NewOption("Yes, run locally (npm run dev)", string(RunModeDev)),
			huh.NewOption("Yes, run with remote bridge (dev-remote)", string(RunModeDevRemote)),
		).
		Value(&runModeStr).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}
	config.RunMode = RunMode(runModeStr)

	runModeLabels := map[string]string{
		string(RunModeNone):      "No",
		string(RunModeDev):       "Yes (local)",
		string(RunModeDevRemote): "Yes (remote)",
	}
	printAnswered(ctx, "Run after creation", runModeLabels[runModeStr])

	return config, nil
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

// PromptForWarehouse shows a picker to select a SQL warehouse.
func PromptForWarehouse(ctx context.Context) (string, error) {
	var warehouses []sql.EndpointInfo
	err := RunWithSpinnerCtx(ctx, "Fetching SQL warehouses...", func() error {
		var fetchErr error
		warehouses, fetchErr = ListSQLWarehouses(ctx)
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch SQL warehouses: %w", err)
	}

	if len(warehouses) == 0 {
		return "", errors.New("no SQL warehouses found. Create one in your workspace first")
	}

	theme := AppkitTheme()

	// Build options with warehouse name and state
	options := make([]huh.Option[string], 0, len(warehouses))
	warehouseNames := make(map[string]string) // id -> name for printing
	for _, wh := range warehouses {
		state := string(wh.State)
		label := fmt.Sprintf("%s (%s)", wh.Name, state)
		options = append(options, huh.NewOption(label, wh.Id))
		warehouseNames[wh.Id] = wh.Name
	}

	var selected string
	err = huh.NewSelect[string]().
		Title("Select SQL Warehouse").
		Description(fmt.Sprintf("%d warehouses available — type to filter", len(warehouses))).
		Options(options...).
		Value(&selected).
		Filtering(true).
		Height(8).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "SQL Warehouse", warehouseNames[selected])
	return selected, nil
}

// PromptForProfile prompts the user to select a CLI profile matching the given workspace host.
// If there is exactly one matching profile, it is returned without prompting.
func PromptForProfile(ctx context.Context, workspaceHost string) (string, error) {
	profiler := profile.GetProfiler(ctx)

	profiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return profile.MatchWorkspaceProfiles(p) && p.Host == workspaceHost
	})
	if err != nil {
		return "", fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(profiles) == 0 {
		return "", nil
	}

	if len(profiles) == 1 {
		return profiles[0].Name, nil
	}

	theme := AppkitTheme()

	// Build options; the first profile is the default.
	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		label := p.Name
		if p.Host != "" {
			label += " (" + p.Host + ")"
		}
		options = append(options, huh.NewOption(label, p.Name))
	}

	selected := profiles[0].Name
	err = huh.NewSelect[string]().
		Title("Select a CLI profile").
		Description(fmt.Sprintf("profile to use for deploy commands (%d available)", len(profiles))).
		Options(options...).
		Value(&selected).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	printAnswered(ctx, "Profile", selected)
	return selected, nil
}

func ResolveProfileForDeploy(ctx context.Context, explicitProfile, workspaceHost string) (string, error) {
	// 1. Explicit --profile flag.
	if explicitProfile != "" {
		return explicitProfile, nil
	}

	// 2. DATABRICKS_CONFIG_PROFILE env var.
	if envProfile := os.Getenv("DATABRICKS_CONFIG_PROFILE"); envProfile != "" {
		return envProfile, nil
	}

	// 3. Prompt for a matching profile.
	if !cmdio.IsPromptSupported(ctx) {
		return "", nil
	}

	return PromptForProfile(ctx, workspaceHost)
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

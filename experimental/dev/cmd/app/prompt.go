package app

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// AppkitTheme returns a custom theme for appkit prompts.
func appkitTheme() *huh.Theme {
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

// CreateProjectConfig holds the configuration gathered from the interactive prompt.
type CreateProjectConfig struct {
	ProjectName  string
	Description  string
	Features     []string
	Dependencies map[string]string // e.g., {"sql_warehouse_id": "abc123"}
	IsNewApp     bool              // true if user chose to create a new app (vs selecting existing)
}

// projectNameValidator validates the project name.
func projectNameValidator(s string) error {
	if s == "" {
		return errors.New("project name is required")
	}

	// Check length constraint (dev- prefix + name <= 30)
	const maxAppNameLength = 30
	const devTargetPrefix = "dev-"
	totalLength := len(devTargetPrefix) + len(s)
	if totalLength > maxAppNameLength {
		maxAllowed := maxAppNameLength - len(devTargetPrefix)
		return fmt.Errorf("name too long (max %d chars)", maxAllowed)
	}

	// Check pattern
	pattern := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	if !pattern.MatchString(s) {
		return errors.New("must start with a letter, use only lowercase letters, numbers, or hyphens")
	}

	return nil
}

// PromptForProjectConfig shows an interactive form to gather project configuration.
// Flow: app picker (if apps exist) -> name -> features -> feature dependencies -> description.
// If preSelectedFeatures is provided, the feature selection prompt is skipped.
// Returns the config and a callback to start app creation (if creating new app).
func PromptForProjectConfig(existingApps []apps.App, preSelectedFeatures []string) (*CreateProjectConfig, error) {
	config := &CreateProjectConfig{
		Dependencies: make(map[string]string),
		Features:     preSelectedFeatures,
	}
	theme := appkitTheme()

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BD2B26")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A"))

	fmt.Println()
	fmt.Println(headerStyle.Render("◆ Create a new Databricks AppKit project"))
	fmt.Println(subtitleStyle.Render("  Full-stack TypeScript • React • Tailwind CSS"))
	fmt.Println()

	// Step 0: App picker (if existing apps are available)
	var selectedApp *apps.App
	if len(existingApps) > 0 {
		selectedApp = promptForExistingApp(existingApps, theme)
	}

	// Track if user is creating a new app
	config.IsNewApp = selectedApp == nil

	// Step 1: Project name (pre-filled if app was selected)
	if selectedApp != nil {
		config.ProjectName = selectedApp.Name
	}

	err := huh.NewInput().
		Title("Project name").
		Description("lowercase letters, numbers, hyphens (max 26 chars)").
		Placeholder("my-app").
		Value(&config.ProjectName).
		Validate(projectNameValidator).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}

	// Step 2: Feature selection (skip if features already provided via flag)
	if len(config.Features) == 0 && len(AvailableFeatures) > 0 {
		options := make([]huh.Option[string], 0, len(AvailableFeatures))
		for _, f := range AvailableFeatures {
			label := f.Name + " - " + f.Description
			options = append(options, huh.NewOption(label, f.ID))
		}

		err = huh.NewMultiSelect[string]().
			Title("Select features").
			Description("space to toggle, enter to confirm").
			Options(options...).
			Value(&config.Features).
			WithTheme(theme).
			Run()
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Prompt for feature dependencies
	deps := CollectDependencies(config.Features)
	for _, dep := range deps {
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
		config.Dependencies[dep.ID] = value
	}

	// Step 4: Description (pre-filled if app was selected)
	defaultDescription := "A Databricks App powered by AppKit"
	if selectedApp != nil && selectedApp.Description != "" {
		config.Description = selectedApp.Description
	} else {
		config.Description = defaultDescription
	}

	err = huh.NewInput().
		Title("Description").
		Placeholder(defaultDescription).
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}

	if config.Description == "" {
		config.Description = defaultDescription
	}

	return config, nil
}

// promptForExistingApp shows a picker for existing apps and returns the selected app.
// Returns nil if user chooses to create a new app.
func promptForExistingApp(existingApps []apps.App, theme *huh.Theme) *apps.App {
	const createNewOption = "__create_new__"

	// Build options: "Create new" first, then existing apps
	options := []huh.Option[string]{
		huh.NewOption("✨ Create a new app", createNewOption),
	}
	appsByName := make(map[string]*apps.App)
	for i := range existingApps {
		app := &existingApps[i]
		label := app.Name
		if app.Description != "" {
			// Truncate long descriptions
			desc := app.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			label += " — " + desc
		}
		options = append(options, huh.NewOption(label, app.Name))
		appsByName[app.Name] = app
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Start from an existing app?").
		Description(fmt.Sprintf("%d of your apps found", len(existingApps))).
		Options(options...).
		Value(&selected).
		WithTheme(theme).
		Run()

	if err != nil || selected == createNewOption {
		return nil
	}

	return appsByName[selected]
}

// RunWithSpinner runs a function while showing a spinner with the given title.
func RunWithSpinner(title string, action func() error) error {
	s := spinner.New(
		spinner.CharSets[14],
		80*time.Millisecond,
		spinner.WithColor("cyan"),
		spinner.WithSuffix(" "+title),
	)
	s.Start()
	err := action()
	s.Stop()
	return err
}

// ListUserApps fetches apps owned by the current user from the workspace.
func ListUserApps(ctx context.Context) ([]apps.App, error) {
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return nil, errors.New("no workspace client available")
	}

	// Get current user to filter by creator
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, err
	}

	iter := w.Apps.List(ctx, apps.ListAppsRequest{})
	allApps, err := listing.ToSlice(ctx, iter)
	if err != nil {
		return nil, err
	}

	// Filter apps by current user
	var userApps []apps.App
	for _, app := range allApps {
		if app.Creator == me.UserName {
			userApps = append(userApps, app)
		}
	}

	return userApps, nil
}

// PromptForAppSelection shows a picker to select an existing app.
// Returns the selected app name or error if cancelled/no apps found.
func PromptForAppSelection(ctx context.Context, title string) (string, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("--name is required in non-interactive mode")
	}

	// Fetch user's apps
	var existingApps []apps.App
	err := RunWithSpinner("Fetching your apps...", func() error {
		var fetchErr error
		existingApps, fetchErr = ListUserApps(ctx)
		return fetchErr
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch apps: %w", err)
	}

	if len(existingApps) == 0 {
		return "", errors.New("no apps found. Create one first with 'databricks apps create <name>'")
	}

	theme := appkitTheme()

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
		Description(fmt.Sprintf("%d of your apps found", len(existingApps))).
		Options(options...).
		Value(&selected).
		WithTheme(theme).
		Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

// PrintSuccess prints a success message after project creation.
func PrintSuccess(projectName, outputDir string, fileCount int) {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFAB00")). // Databricks yellow
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717A")) // Mid-tone gray

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF3621")) // Databricks orange

	fmt.Println()
	fmt.Println(successStyle.Render("✔ Project created successfully!"))
	fmt.Println()
	fmt.Println(dimStyle.Render("  Location: " + outputDir))
	fmt.Println(dimStyle.Render("  Files: " + strconv.Itoa(fileCount)))
	fmt.Println()
	fmt.Println(dimStyle.Render("  Next steps:"))
	fmt.Println()
	fmt.Println(codeStyle.Render("    cd " + projectName))
	fmt.Println(codeStyle.Render("    npm run dev"))
	fmt.Println()
}

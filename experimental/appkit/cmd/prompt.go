package appkit

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// AppkitTheme returns a custom theme for appkit prompts.
func appkitTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Customize colors
	purple := lipgloss.Color("#a855f7")
	gray := lipgloss.Color("#71717a")
	green := lipgloss.Color("#22c55e")

	t.Focused.Title = t.Focused.Title.Foreground(purple).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(gray)

	return t
}

// CreateProjectConfig holds the configuration gathered from the interactive prompt.
type CreateProjectConfig struct {
	ProjectName  string
	Description  string
	Features     []string
	Dependencies map[string]string // e.g., {"sql_warehouse_id": "abc123"}
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
// Flow: name -> features -> feature dependencies -> description.
func PromptForProjectConfig() (*CreateProjectConfig, error) {
	config := &CreateProjectConfig{
		Dependencies: make(map[string]string),
	}
	theme := appkitTheme()

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a855f7")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717a"))

	fmt.Println()
	fmt.Println(headerStyle.Render("◆ Create a new Databricks AppKit project"))
	fmt.Println(subtitleStyle.Render("  Full-stack TypeScript • React • Tailwind CSS"))
	fmt.Println()

	// Step 1: Project name
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

	// Step 2: Feature selection
	if len(AvailableFeatures) > 0 {
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

	// Step 4: Description
	config.Description = "A Databricks App powered by AppKit"
	err = huh.NewInput().
		Title("Description").
		Placeholder("A Databricks App powered by AppKit").
		Value(&config.Description).
		WithTheme(theme).
		Run()
	if err != nil {
		return nil, err
	}

	if config.Description == "" {
		config.Description = "A Databricks App powered by AppKit"
	}

	return config, nil
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

// PrintSuccess prints a success message after project creation.
func PrintSuccess(projectName, outputDir string, fileCount int) {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22c55e")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717a"))

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#38bdf8"))

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

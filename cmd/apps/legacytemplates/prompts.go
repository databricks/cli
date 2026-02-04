package legacytemplates

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdio"
)

// PromptForTemplateType prompts the user to choose between AppKit and framework types.
func PromptForTemplateType(ctx context.Context) (string, error) {
	var choice string
	options := []huh.Option[string]{
		huh.NewOption("AppKit (TypeScript)", "appkit"),
		huh.NewOption("Dash", "dash"),
		huh.NewOption("Flask", "flask"),
		huh.NewOption("Gradio", "gradio"),
		huh.NewOption("Node.js", "nodejs"),
		huh.NewOption("Shiny", "shiny"),
		huh.NewOption("Streamlit", "streamlit"),
	}

	err := huh.NewSelect[string]().
		Title("Select template type").
		Options(options...).
		Value(&choice).
		Height(10).
		WithTheme(prompt.AppkitTheme()).
		Run()
	if err != nil {
		return "", err
	}

	// Get display name for printing
	displayName := choice
	if choice != "appkit" {
		if name, ok := frameworkTypeNames[choice]; ok {
			displayName = name
		}
	} else {
		displayName = "AppKit (TypeScript)"
	}

	prompt.PrintAnswered(ctx, "Template type", displayName)
	return choice, nil
}

// frameworkTypeNames maps framework_type values to their display names.
var frameworkTypeNames = map[string]string{
	"dash":      "Dash",
	"gradio":    "Gradio",
	"streamlit": "Streamlit",
	"flask":     "Flask",
	"shiny":     "Shiny",
	"nodejs":    "Node.js",
}

// PromptForLegacyTemplate prompts the user to select a legacy template.
// If frameworkType is non-empty, only templates matching that framework type are shown.
func PromptForLegacyTemplate(ctx context.Context, templates []AppTemplateManifest, frameworkType string) (*AppTemplateManifest, error) {
	// Filter templates by framework type if specified
	var filteredTemplates []AppTemplateManifest
	var templateIndices []int // Maps filtered index to original index

	if frameworkType != "" {
		for i := range templates {
			if templates[i].FrameworkType == frameworkType {
				filteredTemplates = append(filteredTemplates, templates[i])
				templateIndices = append(templateIndices, i)
			}
		}
	} else {
		filteredTemplates = templates
		templateIndices = make([]int, len(templates))
		for i := range templates {
			templateIndices[i] = i
		}
	}

	if len(filteredTemplates) == 0 {
		return nil, fmt.Errorf("no templates found for framework type: %s", frameworkType)
	}

	options := make([]huh.Option[int], len(filteredTemplates))
	for i := range filteredTemplates {
		tmpl := &filteredTemplates[i]
		// Get framework display name
		frameworkDisplayName := frameworkTypeNames[tmpl.FrameworkType]
		if frameworkDisplayName == "" {
			frameworkDisplayName = tmpl.FrameworkType
		}

		// Build label: "Framework - Name - Description"
		label := frameworkDisplayName + " - " + tmpl.Name
		if tmpl.Description != "" {
			label = frameworkDisplayName + " - " + tmpl.Name + " - " + tmpl.Description
		}
		options[i] = huh.NewOption(label, i)
	}

	var selectedIdx int
	err := huh.NewSelect[int]().
		Title("Select a template").
		Description("Choose from available templates").
		Options(options...).
		Value(&selectedIdx).
		Height(15).
		WithTheme(prompt.AppkitTheme()).
		Run()
	if err != nil {
		return nil, err
	}

	// Map back to original template index
	originalIdx := templateIndices[selectedIdx]
	selectedTemplate := &templates[originalIdx]
	prompt.PrintAnswered(ctx, "Template", selectedTemplate.Name)
	return selectedTemplate, nil
}

// WriteGitignoreIfMissing writes a .gitignore file if it doesn't already exist.
func WriteGitignoreIfMissing(ctx context.Context, destDir, gitignoreContent string) error {
	gitignorePath := filepath.Join(destDir, ".gitignore")

	// Check if .gitignore already exists
	if _, err := os.Stat(gitignorePath); err == nil {
		// .gitignore already exists, skip
		return nil
	}

	// Write the gitignore template
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	cmdio.LogString(ctx, "✓ Created .gitignore")
	return nil
}

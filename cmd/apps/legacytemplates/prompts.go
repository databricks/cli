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

// PromptForTemplateType prompts the user to choose between AppKit and Legacy templates.
func PromptForTemplateType(ctx context.Context) (string, error) {
	var choice string
	options := []huh.Option[string]{
		huh.NewOption("AppKit (TypeScript)", "appkit"),
		huh.NewOption("Legacy template", "legacy"),
	}

	err := huh.NewSelect[string]().
		Title("Select template type").
		Options(options...).
		Value(&choice).
		WithTheme(prompt.AppkitTheme()).
		Run()
	if err != nil {
		return "", err
	}

	prompt.PrintAnswered(ctx, "Template type", choice)
	return choice, nil
}

// PromptForLegacyTemplate prompts the user to select a legacy template.
func PromptForLegacyTemplate(ctx context.Context, templates []AppTemplateManifest) (*AppTemplateManifest, error) {
	options := make([]huh.Option[int], len(templates))
	for i := range templates {
		tmpl := &templates[i]
		label := tmpl.Path
		if tmpl.Manifest.Name != "" {
			label = tmpl.Path + " - " + tmpl.Manifest.Name
			if tmpl.Manifest.Description != "" {
				label = tmpl.Path + " - " + tmpl.Manifest.Name + " - " + tmpl.Manifest.Description
			}
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

	selectedTemplate := &templates[selectedIdx]
	prompt.PrintAnswered(ctx, "Template", selectedTemplate.Path)
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

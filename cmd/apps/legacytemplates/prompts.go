package legacytemplates

import (
	"context"
	"errors"
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

// resourceGetter defines how to get a resource value for a template.
type resourceGetter struct {
	checkRequired func(*AppTemplateManifest) bool
	promptFunc    func(context.Context) (string, error)
	errorMessage  string
}

// getResourceForTemplate is a generic function to get a resource value for a template.
// It checks if the resource is required, uses the provided value if available,
// prompts in interactive mode, or returns an error in non-interactive mode.
func getResourceForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedValue string, isInteractive bool, getter resourceGetter) (string, error) {
	// Check if template requires this resource
	if !getter.checkRequired(tmpl) {
		return "", nil
	}

	// If value was provided via flag, use it
	if providedValue != "" {
		return providedValue, nil
	}

	// In interactive mode, prompt for resource
	if isInteractive {
		value, err := getter.promptFunc(ctx)
		if err != nil {
			return "", err
		}
		return value, nil
	}

	// Non-interactive mode without value - return error
	return "", errors.New(getter.errorMessage)
}

// GetWarehouseIDForTemplate ensures a warehouse ID is available if the template requires one.
func GetWarehouseIDForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedWarehouseID string, isInteractive bool) (string, error) {
	return getResourceForTemplate(ctx, tmpl, providedWarehouseID, isInteractive, resourceGetter{
		checkRequired: RequiresSQLWarehouse,
		promptFunc:    prompt.PromptForWarehouse,
		errorMessage:  "template requires a SQL warehouse. Please provide --warehouse-id",
	})
}

// GetServingEndpointForTemplate ensures a serving endpoint is available if the template requires one.
func GetServingEndpointForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedEndpoint string, isInteractive bool) (string, error) {
	return getResourceForTemplate(ctx, tmpl, providedEndpoint, isInteractive, resourceGetter{
		checkRequired: RequiresServingEndpoint,
		promptFunc:    prompt.PromptForServingEndpoint,
		errorMessage:  "template requires a serving endpoint. Please provide --serving-endpoint",
	})
}

// GetExperimentIDForTemplate ensures an experiment ID is available if the template requires one.
func GetExperimentIDForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedExperimentID string, isInteractive bool) (string, error) {
	return getResourceForTemplate(ctx, tmpl, providedExperimentID, isInteractive, resourceGetter{
		checkRequired: RequiresExperiment,
		promptFunc:    prompt.PromptForExperiment,
		errorMessage:  "template requires an MLflow experiment. Please provide --experiment-id",
	})
}

// GetDatabaseForTemplate ensures database instance and name are available if the template requires them.
// Returns instanceName and databaseName or empty strings if not needed/available.
func GetDatabaseForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedInstanceName, providedDatabaseName string, isInteractive bool) (string, string, error) {
	// Check if template requires a database
	if !RequiresDatabase(tmpl) {
		return "", "", nil
	}

	instanceName := providedInstanceName
	databaseName := providedDatabaseName

	// In interactive mode, prompt for both if not provided
	if isInteractive {
		// Prompt for instance name if not provided
		if instanceName == "" {
			var err error
			instanceName, err = prompt.PromptForDatabaseInstance(ctx)
			if err != nil {
				return "", "", err
			}
		}

		// Prompt for database name if not provided
		if databaseName == "" {
			var err error
			databaseName, err = prompt.PromptForDatabaseName(ctx, instanceName)
			if err != nil {
				return "", "", err
			}
		}

		return instanceName, databaseName, nil
	}

	// Non-interactive mode - both must be provided
	if instanceName == "" || databaseName == "" {
		return "", "", errors.New("template requires a database. Please provide both --instance-name and --database-name")
	}

	return instanceName, databaseName, nil
}

// GetUCVolumeForTemplate ensures a UC volume path is available if the template requires one.
func GetUCVolumeForTemplate(ctx context.Context, tmpl *AppTemplateManifest, providedVolume string, isInteractive bool) (string, error) {
	return getResourceForTemplate(ctx, tmpl, providedVolume, isInteractive, resourceGetter{
		checkRequired: RequiresUCVolume,
		promptFunc:    prompt.PromptForUCVolume,
		errorMessage:  "template requires a Unity Catalog volume. Please provide --uc-volume",
	})
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

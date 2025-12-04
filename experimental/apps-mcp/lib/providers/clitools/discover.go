package clitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/experimental/apps-mcp/lib/detector"
	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// Discover provides workspace context and workflow guidance.
// Returns L1 (flow) always + L2 (target) for detected target types.
func Discover(ctx context.Context, workingDirectory string) (string, error) {
	warehouse, err := middlewares.GetWarehouseEndpoint(ctx)
	if err != nil {
		log.Debugf(ctx, "Failed to get default warehouse (non-fatal): %v", err)
		warehouse = nil
	}

	currentProfile := middlewares.GetDatabricksProfile(ctx)
	profiles := middlewares.GetAvailableProfiles(ctx)

	// run detectors to identify project context
	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, workingDirectory)

	return generateDiscoverGuidance(ctx, warehouse, currentProfile, profiles, detected), nil
}

// generateDiscoverGuidance creates guidance with L1 (flow) + L2 (target) layers.
func generateDiscoverGuidance(ctx context.Context, warehouse *sql.EndpointInfo, currentProfile string, profiles profile.Profiles, detected *detector.DetectedContext) string {
	data := buildTemplateData(warehouse, currentProfile, profiles)

	// L1: always include flow guidance
	result := prompts.MustExecuteTemplate("flow.tmpl", data)

	// L2: inject target-specific guidance for detected target types
	for _, targetType := range detected.TargetTypes {
		templateName := fmt.Sprintf("target_%s.tmpl", targetType)
		if prompts.TemplateExists(templateName) {
			targetContent := prompts.MustExecuteTemplate(templateName, data)
			result += "\n\n" + targetContent
			log.Debugf(ctx, "Injected L2 guidance for target type: %s", targetType)
		} else {
			log.Debugf(ctx, "No L2 template found for target type: %s", targetType)
		}
	}

	// add project context info if detected
	if detected.InProject {
		result += "\n\nDetected project: " + detected.BundleInfo.Name
		if detected.Template != "" {
			result += fmt.Sprintf(" (template: %s)", detected.Template)
		}
	}

	return result
}

func buildTemplateData(warehouse *sql.EndpointInfo, currentProfile string, profiles profile.Profiles) map[string]string {
	workspaceInfo := "Current Workspace Profile: " + currentProfile
	if len(profiles) > 0 {
		var currentHost string
		for _, p := range profiles {
			if p.Name == currentProfile {
				currentHost = p.Host
				if cloud := p.Cloud(); cloud != "" {
					currentHost = fmt.Sprintf("%s (%s)", currentHost, cloud)
				}
				break
			}
		}
		if currentHost != "" {
			workspaceInfo = fmt.Sprintf("Current Workspace Profile: %s - %s", currentProfile, currentHost)
		}
	}

	profilesInfo := ""
	if len(profiles) > 1 {
		profilesInfo = "\n\nAvailable Workspace Profiles:\n"
		for _, p := range profiles {
			marker := ""
			if p.Name == currentProfile {
				marker = " (current)"
			}
			cloud := p.Cloud()
			if cloud != "" {
				profilesInfo += fmt.Sprintf("  - %s: %s (%s)%s\n", p.Name, p.Host, cloud, marker)
			} else {
				profilesInfo += fmt.Sprintf("  - %s: %s%s\n", p.Name, p.Host, marker)
			}
		}
		profilesInfo += "\n  To use a different workspace, add --profile <name> to any command:\n"
		profilesInfo += "    invoke_databricks_cli '--profile prod catalogs list'\n"
	}

	warehouseName := ""
	warehouseID := ""
	if warehouse != nil {
		warehouseName = warehouse.Name
		warehouseID = warehouse.Id
	}

	return map[string]string{
		"WorkspaceInfo": workspaceInfo,
		"WarehouseName": warehouseName,
		"WarehouseID":   warehouseID,
		"ProfilesInfo":  profilesInfo,
		"Profile":       currentProfile,
	}
}

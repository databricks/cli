package clitools

import (
	"context"
	"fmt"

	"github.com/databricks/cli/experimental/apps-mcp/lib/middlewares"
	"github.com/databricks/cli/experimental/apps-mcp/lib/prompts"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// Explore provides guidance on exploring Databricks workspaces and resources.
func Explore(ctx context.Context) (string, error) {
	warehouse, err := middlewares.GetWarehouseEndpoint(ctx)
	if err != nil {
		log.Debugf(ctx, "Failed to get default warehouse (non-fatal): %v", err)
		warehouse = nil
	}

	currentProfile := middlewares.GetDatabricksProfile(ctx)
	profiles := middlewares.GetAvailableProfiles(ctx)

	return generateExploreGuidance(ctx, warehouse, currentProfile, profiles), nil
}

// generateExploreGuidance creates comprehensive guidance for data exploration.
func generateExploreGuidance(ctx context.Context, warehouse *sql.EndpointInfo, currentProfile string, profiles profile.Profiles) string {
	// Build workspace/profile information
	workspaceInfo := "Current Workspace Profile: " + currentProfile
	if len(profiles) > 0 {
		// Find current profile details
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

	// Build available profiles list
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

	// Handle warehouse information (may be nil if lookup failed)
	warehouseName := ""
	warehouseID := ""
	if warehouse != nil {
		warehouseName = warehouse.Name
		warehouseID = warehouse.Id
	}

	// Prepare template data
	data := map[string]string{
		"WorkspaceInfo": workspaceInfo,
		"WarehouseName": warehouseName,
		"WarehouseID":   warehouseID,
		"ProfilesInfo":  profilesInfo,
		"Profile":       currentProfile,
	}

	// Render base explore template
	result := prompts.MustExecuteTemplate("explore.tmpl", data)

	// Get session and check for enabled capabilities
	sess, err := session.GetSession(ctx)
	if err != nil {
		log.Debugf(ctx, "No session found, skipping capability-based instructions: %v", err)
		return result
	}

	capabilities, ok := sess.Get(session.CapabilitiesDataKey)
	if !ok {
		log.Debugf(ctx, "No capabilities set in session")
		return result
	}

	capList, ok := capabilities.([]string)
	if !ok {
		log.Warnf(ctx, "Capabilities is not a string slice, skipping")
		return result
	}

	// Inject additional templates based on enabled capabilities
	for _, cap := range capList {
		switch cap {
		case "apps":
			// Render and append apps template
			appsContent := prompts.MustExecuteTemplate("apps.tmpl", data)
			result = result + "\n\n" + appsContent
			log.Debugf(ctx, "Injected apps instructions based on capability")
		default:
			log.Debugf(ctx, "Unknown capability: %s", cap)
		}
	}

	return result
}

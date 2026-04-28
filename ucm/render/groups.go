package render

import (
	"fmt"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

// resourceGroupsForUcm enumerates UC resource kinds for RenderSummary.
// Skips empty groups so a project with no schemas doesn't print "Schemas:"
// with nothing under it. Mirrors the inline resource-list assembly in
// bundle/render/render_text_output.go::RenderSummary, adapted to ucm's
// resource shapes (no uniform ResourceDescription accessor today).
func resourceGroupsForUcm(u *ucm.Ucm) []ResourceGroup {
	if u == nil {
		return nil
	}
	cfg := &u.Config
	var groups []ResourceGroup

	if g, ok := catalogGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := schemaGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := volumeGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := storageCredentialGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := externalLocationGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := connectionGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := grantGroup(cfg); ok {
		groups = append(groups, g)
	}
	if g, ok := tagValidationRuleGroup(cfg); ok {
		groups = append(groups, g)
	}
	return groups
}

func catalogGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.Catalogs) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.Catalogs))
	for key, c := range cfg.Resources.Catalogs {
		rows = append(rows, ResourceInfo{Key: key, Name: c.Name, URL: c.URL})
	}
	return ResourceGroup{GroupName: "Catalogs", Resources: rows}, true
}

func schemaGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.Schemas) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.Schemas))
	for key, s := range cfg.Resources.Schemas {
		full := s.Name
		if s.CatalogName != "" {
			full = s.CatalogName + "." + s.Name
		}
		rows = append(rows, ResourceInfo{Key: key, Name: full, URL: s.URL})
	}
	return ResourceGroup{GroupName: "Schemas", Resources: rows}, true
}

func volumeGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.Volumes) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.Volumes))
	for key, v := range cfg.Resources.Volumes {
		full := v.Name
		if v.CatalogName != "" && v.SchemaName != "" {
			full = v.CatalogName + "." + v.SchemaName + "." + v.Name
		}
		rows = append(rows, ResourceInfo{Key: key, Name: full, URL: v.URL})
	}
	return ResourceGroup{GroupName: "Volumes", Resources: rows}, true
}

func storageCredentialGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.StorageCredentials) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.StorageCredentials))
	for key, sc := range cfg.Resources.StorageCredentials {
		rows = append(rows, ResourceInfo{Key: key, Name: sc.Name, URL: sc.URL})
	}
	return ResourceGroup{GroupName: "Storage credentials", Resources: rows}, true
}

func externalLocationGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.ExternalLocations) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.ExternalLocations))
	for key, el := range cfg.Resources.ExternalLocations {
		rows = append(rows, ResourceInfo{Key: key, Name: el.Name, URL: el.URL})
	}
	return ResourceGroup{GroupName: "External locations", Resources: rows}, true
}

func connectionGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.Connections) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.Connections))
	for key, conn := range cfg.Resources.Connections {
		rows = append(rows, ResourceInfo{Key: key, Name: conn.Name, URL: conn.URL})
	}
	return ResourceGroup{GroupName: "Connections", Resources: rows}, true
}

func grantGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.Grants) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.Grants))
	for key, g := range cfg.Resources.Grants {
		// Grants have no workspace URL; summarise securable + principal.
		name := fmt.Sprintf("%s %s -> %s", g.Securable.Type, g.Securable.Name, g.Principal)
		rows = append(rows, ResourceInfo{Key: key, Name: name})
	}
	return ResourceGroup{GroupName: "Grants", Resources: rows}, true
}

func tagValidationRuleGroup(cfg *config.Root) (ResourceGroup, bool) {
	if len(cfg.Resources.TagValidationRules) == 0 {
		return ResourceGroup{}, false
	}
	rows := make([]ResourceInfo, 0, len(cfg.Resources.TagValidationRules))
	for key := range cfg.Resources.TagValidationRules {
		rows = append(rows, ResourceInfo{Key: key, Name: key})
	}
	return ResourceGroup{GroupName: "Tag validation rules", Resources: rows}, true
}

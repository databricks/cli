package installer

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

const (
	legacyAIDevKitServerName    = "databricks"
	legacyAIDevKitMCPServersKey = "mcpServers"
	legacyAIDevKitServersKey    = "servers"
	legacyAIDevKitConfigMaxSize = 1 << 20
	legacyAIDevKitSkillDetail   = "legacy skill directory"
)

var promptLegacyAIDevKitRemoval = defaultPromptLegacyAIDevKitRemoval

var legacyAIDevKitSkillNames = map[string]struct{}{
	"databricks-app-python":           {},
	"databricks-apps-python":          {},
	"databricks-asset-bundles":        {},
	"databricks-bundles":              {},
	"databricks-config":               {},
	"databricks-lakebase-autoscale":   {},
	"databricks-lakebase-provisioned": {},
	"databricks-parsing":              {},
}

type legacyAIDevKitFinding struct {
	path   string
	detail string
}

type legacyAIDevKitScanBase struct {
	root string
}

// warnLegacyAIDevKitArtifacts prints cleanup guidance for stale AI Dev Kit files.
func warnLegacyAIDevKitArtifacts(ctx context.Context, scope string) {
	findings := detectLegacyAIDevKitArtifacts(ctx, scope)
	if len(findings) == 0 {
		return
	}

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, cmdio.Yellow(ctx, "Found legacy Databricks AI Dev Kit artifacts."))
	cmdio.LogString(ctx, "These can load the old Databricks MCP server or duplicate skills alongside Databricks AI skills:")
	for _, finding := range findings {
		cmdio.LogString(ctx, "  - "+finding.path+" ("+finding.detail+")")
	}
	cmdio.LogString(ctx, "Remove the databricks MCP server entry from listed MCP config files after confirming it is no longer needed.")
	cmdio.LogString(ctx, "Remove legacy AI Dev Kit skill directories listed above after confirming they are no longer needed.")
	if scope == ScopeProject {
		cmdio.LogString(ctx, "If the legacy MCP server was installed globally, run 'databricks aitools install --scope=global' to migrate global usage too.")
	}
	if !cmdio.IsPromptSupported(ctx) {
		return
	}
	remove, err := promptLegacyAIDevKitRemoval(ctx, findings)
	if err != nil {
		cmdio.LogString(ctx, "Could not prompt for legacy AI Dev Kit cleanup: "+err.Error())
		return
	}
	if !remove {
		return
	}
	if err := removeLegacyAIDevKitArtifacts(findings); err != nil {
		cmdio.LogString(ctx, "Failed to remove legacy Databricks AI Dev Kit artifacts: "+err.Error())
		return
	}
	cmdio.LogString(ctx, "Removed legacy Databricks AI Dev Kit artifacts.")
}

// defaultPromptLegacyAIDevKitRemoval asks whether to remove detected legacy artifacts.
func defaultPromptLegacyAIDevKitRemoval(_ context.Context, _ []legacyAIDevKitFinding) (bool, error) {
	var remove bool
	err := huh.NewConfirm().
		Title("Remove legacy Databricks AI Dev Kit artifacts?").
		Description("This removes only the listed legacy skill directories and databricks MCP server entries.").
		Affirmative("Remove").
		Negative("Keep").
		Value(&remove).
		Run()
	return remove, err
}

// hasLegacyAIDevKitArtifacts returns true when AI Dev Kit migration targets exist.
func hasLegacyAIDevKitArtifacts(ctx context.Context, scope string) bool {
	return len(detectLegacyAIDevKitArtifacts(ctx, scope)) > 0
}

// detectLegacyAIDevKitArtifacts returns source-confirmed AI Dev Kit cleanup targets.
func detectLegacyAIDevKitArtifacts(ctx context.Context, scope string) []legacyAIDevKitFinding {
	var findings []legacyAIDevKitFinding
	seen := make(map[string]struct{})
	for _, base := range legacyAIDevKitScanBases(ctx, scope) {
		for _, finding := range legacyAIDevKitMCPFindings(base.root) {
			addLegacyAIDevKitFinding(&findings, seen, finding)
		}
		for _, finding := range legacyAIDevKitSkillFindings(base.root) {
			addLegacyAIDevKitFinding(&findings, seen, finding)
		}
	}
	slices.SortFunc(findings, func(a, b legacyAIDevKitFinding) int {
		if a.path == b.path {
			return cmp.Compare(a.detail, b.detail)
		}
		return cmp.Compare(a.path, b.path)
	})
	return findings
}

// legacyAIDevKitScanBases returns filesystem roots relevant to the install scope.
func legacyAIDevKitScanBases(ctx context.Context, scope string) []legacyAIDevKitScanBase {
	var bases []legacyAIDevKitScanBase
	homeDir, err := env.UserHomeDir(ctx)
	if err == nil {
		bases = append(bases, legacyAIDevKitScanBase{root: homeDir})
	}
	if scope == ScopeProject {
		cwd, err := os.Getwd()
		if err == nil && cwd != homeDir {
			bases = append(bases, legacyAIDevKitScanBase{root: cwd})
		}
	}
	return bases
}

// addLegacyAIDevKitFinding appends a finding unless the path and detail already appeared.
func addLegacyAIDevKitFinding(findings *[]legacyAIDevKitFinding, seen map[string]struct{}, finding legacyAIDevKitFinding) {
	key := finding.path + "\x00" + finding.detail
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*findings = append(*findings, finding)
}

// legacyAIDevKitMCPFindings returns MCP config entries using the old Databricks server.
func legacyAIDevKitMCPFindings(root string) []legacyAIDevKitFinding {
	// Source paths come from the legacy AI Dev Kit installer:
	// https://github.com/databricks-solutions/ai-dev-kit/blob/1b75b9e7191a47fde015376f69f06aaee52f6c65/install.sh#L1516-L1562
	paths := []string{
		filepath.Join(root, ".claude.json"),
		filepath.Join(root, ".claude", "mcp.json"),
		filepath.Join(root, ".mcp.json"),
		filepath.Join(root, ".cursor", "mcp.json"),
		filepath.Join(root, ".vscode", "mcp.json"),
	}
	var findings []legacyAIDevKitFinding
	for _, path := range paths {
		for _, key := range legacyAIDevKitMCPKeys(path) {
			findings = append(findings, legacyAIDevKitFinding{path: path, detail: key})
		}
	}
	return findings
}

// legacyAIDevKitMCPKeys returns legacy Databricks MCP server keys in a config file.
func legacyAIDevKitMCPKeys(path string) []string {
	content, _, ok, err := readLegacyAIDevKitConfig(path)
	if err != nil || !ok {
		return nil
	}
	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
		Servers    map[string]json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil
	}
	var keys []string
	if _, ok := config.MCPServers[legacyAIDevKitServerName]; ok {
		keys = append(keys, legacyAIDevKitMCPServersKey+"."+legacyAIDevKitServerName)
	}
	if _, ok := config.Servers[legacyAIDevKitServerName]; ok {
		keys = append(keys, legacyAIDevKitServersKey+"."+legacyAIDevKitServerName)
	}
	return keys
}

// legacyAIDevKitSkillFindings returns legacy skill directories under agent skill roots.
func legacyAIDevKitSkillFindings(root string) []legacyAIDevKitFinding {
	// Legacy skill directories were written under agent skill roots by AI Dev Kit:
	// https://github.com/databricks-solutions/ai-dev-kit/blob/1b75b9e7191a47fde015376f69f06aaee52f6c65/install.sh#L1113-L1120
	skillRoots := []string{
		filepath.Join(root, ".claude", "skills"),
		filepath.Join(root, ".cursor", "skills"),
		filepath.Join(root, ".github", "skills"),
	}
	legacyNames := slices.Sorted(maps.Keys(legacyAIDevKitSkillNames))
	var findings []legacyAIDevKitFinding
	for _, skillRoot := range skillRoots {
		for _, name := range legacyNames {
			path := filepath.Join(skillRoot, name)
			if legacyAIDevKitSkillExists(path) {
				findings = append(findings, legacyAIDevKitFinding{path: path, detail: legacyAIDevKitSkillDetail})
			}
		}
	}
	return findings
}

// removeLegacyAIDevKitArtifacts removes only the detected legacy cleanup targets.
func removeLegacyAIDevKitArtifacts(findings []legacyAIDevKitFinding) error {
	mcpKeysByPath := make(map[string][]string)
	for _, finding := range findings {
		switch finding.detail {
		case legacyAIDevKitSkillDetail:
			if err := os.RemoveAll(finding.path); err != nil {
				return fmt.Errorf("remove legacy skill %s: %w", finding.path, err)
			}
		case legacyAIDevKitMCPServersKey + "." + legacyAIDevKitServerName, legacyAIDevKitServersKey + "." + legacyAIDevKitServerName:
			mcpKeysByPath[finding.path] = append(mcpKeysByPath[finding.path], finding.detail)
		}
	}
	for path, keys := range mcpKeysByPath {
		if err := removeLegacyAIDevKitMCPKeys(path, keys); err != nil {
			return err
		}
	}
	return nil
}

// removeLegacyAIDevKitMCPKeys deletes selected Databricks MCP server keys.
func removeLegacyAIDevKitMCPKeys(path string, keys []string) error {
	content, mode, ok, err := readLegacyAIDevKitConfig(path)
	if err != nil {
		return fmt.Errorf("read legacy MCP config %s: %w", path, err)
	}
	if !ok {
		return nil
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(content, &config); err != nil {
		return nil
	}
	changed := false
	for _, key := range keys {
		switch key {
		case legacyAIDevKitMCPServersKey + "." + legacyAIDevKitServerName:
			changed = removeLegacyAIDevKitServer(config, legacyAIDevKitMCPServersKey) || changed
		case legacyAIDevKitServersKey + "." + legacyAIDevKitServerName:
			changed = removeLegacyAIDevKitServer(config, legacyAIDevKitServersKey) || changed
		}
	}
	if !changed {
		return nil
	}
	updated, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode legacy MCP config %s: %w", path, err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(path, updated, mode.Perm()); err != nil {
		return fmt.Errorf("write legacy MCP config %s: %w", path, err)
	}
	return nil
}

// removeLegacyAIDevKitServer removes the Databricks server from a top-level map.
func removeLegacyAIDevKitServer(config map[string]json.RawMessage, topKey string) bool {
	rawServers, ok := config[topKey]
	if !ok {
		return false
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(rawServers, &servers); err != nil {
		return false
	}
	if _, ok := servers[legacyAIDevKitServerName]; !ok {
		return false
	}
	delete(servers, legacyAIDevKitServerName)
	if len(servers) == 0 {
		delete(config, topKey)
		return true
	}
	updated, err := json.Marshal(servers)
	if err != nil {
		return false
	}
	config[topKey] = updated
	return true
}

// readLegacyAIDevKitConfig reads small regular JSON config files only.
func readLegacyAIDevKitConfig(path string) ([]byte, fs.FileMode, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > legacyAIDevKitConfigMaxSize {
		return nil, 0, false, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, legacyAIDevKitConfigMaxSize+1))
	if err != nil {
		return nil, 0, false, err
	}
	if len(content) > legacyAIDevKitConfigMaxSize {
		return nil, 0, false, nil
	}
	return content, info.Mode(), true, nil
}

// legacyAIDevKitSkillExists returns true for legacy skill dirs or symlinks.
func legacyAIDevKitSkillExists(path string) bool {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode()&os.ModeSymlink != 0
}

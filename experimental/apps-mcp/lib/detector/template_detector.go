package detector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TemplateDetector detects the template type from project files.
type TemplateDetector struct{}

// packageJSON represents relevant parts of package.json.
type packageJSON struct {
	Name         string            `json:"name"`
	Dependencies map[string]string `json:"dependencies"`
}

// Detect identifies the template type from project configuration files.
func (d *TemplateDetector) Detect(ctx context.Context, workDir string, detected *DetectedContext) error {
	// check for appkit-typescript (package.json with specific markers)
	if template := d.detectFromPackageJSON(workDir); template != "" {
		detected.Template = template
		return nil
	}

	// check for python template (pyproject.toml)
	if template := d.detectFromPyproject(workDir); template != "" {
		detected.Template = template
		return nil
	}

	return nil
}

func (d *TemplateDetector) detectFromPackageJSON(workDir string) string {
	pkgPath := filepath.Join(workDir, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return ""
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// check for appkit markers
	if _, hasAppkit := pkg.Dependencies["@databricks/sql"]; hasAppkit {
		return "appkit-typescript"
	}

	// check for trpc which is used in appkit
	for dep := range pkg.Dependencies {
		if strings.HasPrefix(dep, "@trpc/") {
			return "appkit-typescript"
		}
	}

	return ""
}

func (d *TemplateDetector) detectFromPyproject(workDir string) string {
	pyprojectPath := filepath.Join(workDir, "pyproject.toml")

	if _, err := os.Stat(pyprojectPath); err == nil {
		// pyproject.toml exists - likely python template
		return "python"
	}

	return ""
}

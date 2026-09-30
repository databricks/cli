package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/libs/apps/manifest"
	"go.yaml.in/yaml/v3"
)

type appYAML struct {
	Env []struct {
		Name  string  `yaml:"name"`
		Value *string `yaml:"value"`
	} `yaml:"env"`
}

type bundleYAML struct {
	Resources struct {
		Apps map[string]struct {
			UserAPIScopes []string `yaml:"user_api_scopes"`
			Resources     []struct {
				Name string `yaml:"name"`
			} `yaml:"resources"`
		} `yaml:"apps"`
	} `yaml:"resources"`
}

// ValidateAuthModes checks that resources accessed on behalf of the user are consistent
// with the manifest. A resource is treated as accessed on behalf of the user when app.yaml
// sets one of its env vars to a literal value and databricks.yml does not bind it.
// Such a resource must not be app-only, and its scope must be in user_api_scopes.
// Projects without appkit.plugins.json, app.yaml, or databricks.yml are not checked.
func ValidateAuthModes(projectPath string) error {
	if !manifest.HasManifest(projectPath) {
		return nil
	}
	m, err := manifest.Load(projectPath)
	if err != nil {
		return err
	}

	var app appYAML
	if ok, err := readYAML(filepath.Join(projectPath, "app.yaml"), &app); !ok {
		return err
	}
	// ponytail: reads only the root databricks.yml, include files are not followed.
	var bundle bundleYAML
	if ok, err := readYAML(filepath.Join(projectPath, "databricks.yml"), &bundle); !ok {
		return err
	}

	bound := make(map[string]bool)
	var scopes []string
	for _, a := range bundle.Resources.Apps {
		for _, r := range a.Resources {
			bound[r.Name] = true
		}
		scopes = append(scopes, a.UserAPIScopes...)
	}

	byEnv := make(map[string]manifest.Resource)
	for _, p := range m.GetPlugins() {
		for _, r := range append(p.Resources.Required, p.Resources.Optional...) {
			for _, f := range r.Fields {
				if f.Env != "" {
					byEnv[f.Env] = r
				}
			}
		}
	}

	var errs []error
	checked := make(map[string]bool)
	for _, e := range app.Env {
		r, ok := byEnv[e.Name]
		if !ok || e.Value == nil || bound[r.Key()] || checked[r.Key()] {
			continue
		}
		checked[r.Key()] = true
		switch {
		case r.AppOnly:
			errs = append(errs, fmt.Errorf("resource %q must be bound to the app's service principal in databricks.yml, but app.yaml sets %s to a literal value", r.Key(), e.Name))
		case r.Scope != "" && !slices.Contains(scopes, r.Scope):
			errs = append(errs, fmt.Errorf("resource %q is accessed on behalf of the user but its scope %q is missing from user_api_scopes in databricks.yml", r.Key(), r.Scope))
		}
	}
	return errors.Join(errs...)
}

// readYAML parses path into v. Returns false with a nil error if the file does not exist.
func readYAML(path string, v any) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return true, nil
}

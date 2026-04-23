package loader

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
)

type processRootIncludes struct{}

// ProcessRootIncludes expands every glob in Root.Include into matching files
// and loads each through ProcessInclude.
func ProcessRootIncludes() ucm.Mutator {
	return &processRootIncludes{}
}

func (m *processRootIncludes) Name() string {
	return "ProcessRootIncludes"
}

// globChars enumerates the metacharacters recognized by [filepath.Glob]; a
// ucm root path that contains any of them is rejected outright.
var globChars = []string{"*", "?", "[", "]", "^"}

func hasGlobCharacters(path string) (string, bool) {
	for _, c := range globChars {
		if strings.Contains(path, c) {
			return c, true
		}
	}
	return "", false
}

func (m *processRootIncludes) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	var out []ucm.Mutator

	// Root ucm.yml filenames are already loaded — skip them if they happen to
	// match an include glob.
	seen := map[string]bool{}
	for _, file := range config.FileNames {
		seen[file] = true
	}

	var files []string
	var diags diag.Diagnostics

	// Reject glob metacharacters in the root path: filepath.Glob would parse
	// them as patterns and produce surprising behavior.
	if char, ok := hasGlobCharacters(u.RootPath); ok {
		return diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Ucm root path contains glob pattern characters",
			Detail:   fmt.Sprintf("The path to the ucm root %s contains glob pattern character %q. Please remove the character from this path to use ucm commands.", u.RootPath, char),
		})
	}

	for _, entry := range u.Config.Include {
		if filepath.IsAbs(entry) {
			return diag.Errorf("%s: includes must be relative paths", entry)
		}

		matches, err := filepath.Glob(filepath.Join(u.RootPath, entry))
		if err != nil {
			return diag.FromErr(err)
		}

		// A literal (non-glob) entry with no match is an error; a glob entry
		// that matched nothing only logs a warning (matches DAB behavior).
		if len(matches) == 0 && !strings.ContainsAny(entry, "*?[") {
			return diag.Errorf("%s defined in 'include' section does not match any files", entry)
		}

		var includes []string
		for i, match := range matches {
			rel, err := filepath.Rel(u.RootPath, match)
			if err != nil {
				return diag.FromErr(err)
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = true
			if ext := filepath.Ext(rel); ext != ".yaml" && ext != ".yml" && ext != ".json" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "Files in the 'include' configuration section must be YAML or JSON files.",
					Detail:    fmt.Sprintf("The file %s in the 'include' configuration section is not a YAML or JSON file, and only such files are supported.", rel),
					Locations: u.Config.GetLocations(fmt.Sprintf("include[%d]", i)),
				})
				continue
			}
			includes = append(includes, rel)
		}

		if len(diags) > 0 {
			return diags
		}

		slices.Sort(includes)
		files = append(files, includes...)
		for _, include := range includes {
			out = append(out, ProcessInclude(filepath.Join(u.RootPath, include), include))
		}
	}

	u.Config.Include = files
	ucm.ApplySeqContext(ctx, u, out...)
	return nil
}

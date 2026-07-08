package generate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnIfNotIncluded warns when the freshly generated configuration file is not
// matched by any pattern in the root databricks.yml 'include' section. Such a
// file is written to disk but silently ignored during deploy, so without a
// matching 'include' entry the generate command is effectively a no-op.
func warnIfNotIncluded(ctx context.Context, b *bundle.Bundle, filename string) {
	// b.Config.Include has been overwritten with the expanded list of already
	// loaded files by ProcessRootIncludes, so re-read the root configuration
	// file to recover the raw (unexpanded) include patterns.
	rootFile, err := config.FileNames.FindInPath(b.BundleRootPath)
	if err != nil {
		log.Debugf(ctx, "Skipping include check: %v", err)
		return
	}

	root, diags := config.Load(rootFile)
	if diags.HasError() {
		log.Debugf(ctx, "Skipping include check: failed to load %s: %v", rootFile, diags.Error())
		return
	}

	absFilename, err := filepath.Abs(filename)
	if err != nil {
		log.Debugf(ctx, "Skipping include check: %v", err)
		return
	}

	if isIncluded(b.BundleRootPath, root.Include, absFilename) {
		return
	}

	rel, err := filepath.Rel(b.BundleRootPath, absFilename)
	if err != nil {
		rel = absFilename
	}
	rel = filepath.ToSlash(rel)

	// Suggest a glob that covers the directory the file was written to.
	suggestion := filepath.ToSlash(filepath.Join(filepath.Dir(rel), "*.yml"))

	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Generated configuration is not included in the bundle",
		Detail: fmt.Sprintf(""+
			"The file %s is not matched by any pattern in the 'include' section of databricks.yml,\n"+
			"so it will not be deployed. Add a matching entry to the 'include' section, for example:\n"+
			"\n"+
			"include:\n"+
			"  - %s",
			rel, suggestion),
	})
}

// isIncluded reports whether absFilename is matched by any of the include
// patterns, resolved the same way as ProcessRootIncludes: each pattern is
// anchored to the bundle root and expanded with filepath.Glob against the
// files on disk.
func isIncluded(bundleRootPath string, include []string, absFilename string) bool {
	for _, entry := range include {
		// Includes must be relative paths; the loader errors on absolute ones.
		if filepath.IsAbs(entry) {
			continue
		}

		matches, err := filepath.Glob(filepath.Join(bundleRootPath, entry))
		if err != nil {
			continue
		}

		for _, match := range matches {
			absMatch, err := filepath.Abs(match)
			if err != nil {
				continue
			}
			if absMatch == absFilename {
				return true
			}
		}
	}

	return false
}

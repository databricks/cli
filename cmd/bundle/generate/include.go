package generate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// warnIfNotIncluded warns when the freshly generated configuration file is not
// matched by any pattern in the root databricks.yml 'include' section. Such a
// file is written to disk but silently ignored during deploy, so without a
// matching 'include' entry the generate command is effectively a no-op.
func warnIfNotIncluded(ctx context.Context, b *bundle.Bundle, filename string) {
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		log.Debugf(ctx, "Skipping include check: %v", err)
		return
	}

	if b.IsFileIncluded(absFilename) {
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

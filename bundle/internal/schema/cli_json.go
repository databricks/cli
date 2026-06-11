package main

import (
	"fmt"

	"github.com/databricks/cli/internal/clijson"
)

// parseCliJSON reads .codegen/cli.json via the shared clijson.Parse and returns
// its schema graph keyed by SDK type name (e.g. "jobs.JobSettings"). The
// annotation parser matches Go SDK types against these keys directly; refs
// between schemas are stored as bare type names, so no $ref reconstruction is
// needed.
func parseCliJSON(path string) (map[string]*clijson.SchemaJSON, error) {
	doc, err := clijson.Parse(path)
	if err != nil {
		return nil, err
	}

	if len(doc.Schemas) == 0 {
		return nil, fmt.Errorf("no schemas found in %s", path)
	}

	return doc.Schemas, nil
}

package lsp

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// ResourceEntry represents a resource definition found in YAML.
type ResourceEntry struct {
	Type     string // e.g., "jobs", "pipelines"
	Key      string // e.g., "my_etl_job"
	KeyRange Range  // position of the key in the YAML file
	Path     string // e.g., "resources.jobs.my_etl_job"
}

// IndexResources walks a parsed YAML dyn.Value and finds all resource definitions
// under "resources.<type>.<key>".
func IndexResources(doc *Document) []ResourceEntry {
	if !doc.Value.IsValid() {
		return nil
	}

	resources := doc.Value.Get("resources")
	if resources.Kind() != dyn.KindMap {
		return nil
	}

	m, ok := resources.AsMap()
	if !ok {
		return nil
	}

	var entries []ResourceEntry
	for _, typePair := range m.Pairs() {
		resourceType := typePair.Key.MustString()
		typeVal := typePair.Value
		if typeVal.Kind() != dyn.KindMap {
			continue
		}

		typeMap, ok := typeVal.AsMap()
		if !ok {
			continue
		}

		for _, resPair := range typeMap.Pairs() {
			key := resPair.Key.MustString()
			keyLoc := resPair.Key.Location()

			// dyn.Location uses 1-based line/column; LSP uses 0-based.
			startLine := max(keyLoc.Line-1, 0)
			startChar := max(keyLoc.Column-1, 0)

			entries = append(entries, ResourceEntry{
				Type: resourceType,
				Key:  key,
				KeyRange: Range{
					Start: Position{Line: startLine, Character: startChar},
					End:   Position{Line: startLine, Character: startChar + len(key)},
				},
				Path: fmt.Sprintf("resources.%s.%s", resourceType, key),
			})
		}
	}

	return entries
}

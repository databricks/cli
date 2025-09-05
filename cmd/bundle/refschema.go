package bundle

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
	"github.com/databricks/cli/libs/utils"
	"github.com/spf13/cobra"
)

func newRefSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ref-schema",
		Short: "Dump all relevant fields all bundle resources",
		Long: `Dumps all available fields for each resource type by walking the relevant types. Each line is a path to a field, type and set of tags:
- INPUT: field is present in bundle config.
- STATE: field is present in the state of the resource (direct deployment only).
- REMOTE: field is present in the remote state of the resource (direct deployment only).
- ALL: shortcut for all three.
`,
		Args:   root.NoArgs,
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return dumpRemoteSchemas()
	}

	return cmd
}

// dumpRemoteSchemas walks through all supported resources and dumps their schema fields.
func dumpRemoteSchemas() error {
	adapters, err := tnresources.InitAll(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %w", err)
	}

	for _, resourceName := range utils.SortedKeys(adapters) {
		adapter := adapters[resourceName]

		// path -> typeLabel -> set of sources
		pathTypes := make(map[string]map[string]map[string]struct{})

		collect := func(root reflect.Type, source string) error {
			return structwalk.WalkType(root, func(path *structpath.PathNode, typ reflect.Type) bool {
				if path == nil {
					return true
				}
				p := convertArraysToIndexed(path.String())
				p = strings.TrimPrefix(p, ".")
				t := fmt.Sprint(typ)
				byType, ok := pathTypes[p]
				if !ok {
					byType = make(map[string]map[string]struct{})
					pathTypes[p] = byType
				}
				sources, ok := byType[t]
				if !ok {
					sources = make(map[string]struct{})
					byType[t] = sources
				}
				sources[source] = struct{}{}
				return true
			})
		}

		if err := collect(adapter.InputConfigType(), "INPUT"); err != nil {
			return fmt.Errorf("failed to walk input type for %s: %w", resourceName, err)
		}
		if err := collect(adapter.ConfigType(), "STATE"); err != nil {
			return fmt.Errorf("failed to walk config type for %s: %w", resourceName, err)
		}
		if err := collect(adapter.RemoteType(), "REMOTE"); err != nil {
			return fmt.Errorf("failed to walk remote type for %s: %w", resourceName, err)
		}

		var lines []string
		for _, p := range utils.SortedKeys(pathTypes) {
			byType := pathTypes[p]
			for _, t := range utils.SortedKeys(byType) {
				info := formatTags(byType[t])
				lines = append(lines, fmt.Sprintf("resources.%s.*.%s\t%s\t%s", resourceName, p, t, info))
			}
		}

		sort.Strings(lines)
		for _, l := range lines {
			fmt.Println(l)
		}
	}

	return nil
}

func formatTags(sources map[string]struct{}) string {
	if len(sources) == 3 {
		return "ALL"
	}
	return strings.Join(utils.SortedKeys(sources), "\t")
}

// convertArraysToIndexed converts array patterns like "tasks[*]" to "tasks[0]" format.
func convertArraysToIndexed(path string) string {
	// Replace patterns like "field[*]" with "field[0]"
	result := strings.ReplaceAll(path, "[*]", "[0]")
	return result
}

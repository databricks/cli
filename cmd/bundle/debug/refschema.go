package debug

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/utils"
	"github.com/spf13/cobra"
)

func NewRefSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refschema",
		Short: "Dump all relevant fields all bundle resources",
		Long: `Dumps all available fields for each resource type by walking the relevant types. Each line is a path to a field, type and set of tags:
- INPUT: field is present in bundle config.
- STATE: field is present in the state of the resource (direct deployment only).
- REMOTE: field is present in the remote state of the resource (direct deployment only).
- ALL: shortcut for all three.
`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return dumpRemoteSchemas(cmd.OutOrStdout())
	}

	return cmd
}

// dumpRemoteSchemas walks through all supported resources and dumps their schema fields.
func dumpRemoteSchemas(out io.Writer) error {
	// path -> typeLabel -> set of sources
	pathTypes := make(map[string]map[string]map[string]struct{})

	collect := func(fullPath string, typ reflect.Type, source string) {
		t := strings.ReplaceAll(fmt.Sprint(typ), "interface {}", "any")
		byType, ok := pathTypes[fullPath]
		if !ok {
			byType = make(map[string]map[string]struct{})
			pathTypes[fullPath] = byType
		}
		sources, ok := byType[t]
		if !ok {
			sources = make(map[string]struct{})
			byType[t] = sources
		}
		sources[source] = struct{}{}
	}

	// Walk config.Resources for INPUT - this naturally gives us the map structure
	// including entries like "resources.volumes" and "resources.volumes.*"
	err := structwalk.WalkType(reflect.TypeOf(config.Resources{}), func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
		if path == nil {
			return true
		}
		p := strings.TrimPrefix(path.String(), ".")
		collect("resources."+p, typ, "INPUT")
		return true
	})
	if err != nil {
		return fmt.Errorf("failed to walk config.Resources: %w", err)
	}

	// Use adapters for STATE and REMOTE (and sub-resource INPUT like permissions/grants)
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %w", err)
	}

	for _, resourceName := range utils.SortedKeys(adapters) {
		adapter := adapters[resourceName]

		var resourcePrefix string
		isSubResource := strings.Contains(resourceName, ".")

		if isSubResource {
			// "jobs.permissions" -> "resources.jobs.*.permissions"
			resourcePrefix = "resources." + strings.ReplaceAll(resourceName, ".", ".*.")
		} else {
			resourcePrefix = "resources." + resourceName + ".*"
		}

		collectFromType := func(root reflect.Type, source string) error {
			return structwalk.WalkType(root, func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
				var fullPath string
				if path == nil {
					fullPath = resourcePrefix
				} else {
					fullPath = resourcePrefix + "." + strings.TrimPrefix(path.String(), ".")
				}
				collect(fullPath, typ, source)
				return true
			})
		}

		// For sub-resources (permissions, grants), collect INPUT since they're not in config.Resources
		if isSubResource {
			if err := collectFromType(adapter.InputConfigType(), "INPUT"); err != nil {
				return fmt.Errorf("failed to walk input type for %s: %w", resourceName, err)
			}
		}

		if err := collectFromType(adapter.StateType(), "STATE"); err != nil {
			return fmt.Errorf("failed to walk state type for %s: %w", resourceName, err)
		}
		if err := collectFromType(adapter.RemoteType(), "REMOTE"); err != nil {
			return fmt.Errorf("failed to walk remote type for %s: %w", resourceName, err)
		}
	}

	// Output all collected paths
	var lines []string
	for _, p := range utils.SortedKeys(pathTypes) {
		byType := pathTypes[p]
		for _, t := range utils.SortedKeys(byType) {
			info := formatTags(byType[t])
			lines = append(lines, fmt.Sprintf("%s\t%s\t%s\n", p, t, info))
		}
	}

	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprint(out, l)
	}

	return nil
}

func formatTags(sources map[string]struct{}) string {
	if len(sources) == 3 {
		return "ALL"
	}
	return strings.Join(utils.SortedKeys(sources), "\t")
}

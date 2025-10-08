package debug

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

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
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %w", err)
	}

	for _, resourceName := range utils.SortedKeys(adapters) {
		adapter := adapters[resourceName]

		var resourcePrefix string

		if strings.Contains(resourceName, ".") {
			// "jobs.permissions" -> "resources.jobs.*.permissions"
			resourcePrefix = "resources." + strings.ReplaceAll(resourceName, ".", ".*.")
		} else {
			resourcePrefix = "resources." + resourceName + ".*"
		}

		// TODO: fields with bundle: tag has variety of behaviors
		// id is REMOTE but it shows up in inputType
		// "url" is remote on some resources
		// "modified_status" in internal

		// path -> typeLabel -> set of sources
		pathTypes := make(map[string]map[string]map[string]struct{})

		collect := func(root reflect.Type, source string) error {
			return structwalk.WalkType(root, func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
				if path == nil {
					return true
				}

				p := path.String()
				p = strings.TrimPrefix(p, ".")
				t := strings.ReplaceAll(fmt.Sprint(typ), "interface {}", "any")
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
		if err := collect(adapter.StateType(), "STATE"); err != nil {
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
				lines = append(lines, fmt.Sprintf("%s.%s\t%s\t%s\n", resourcePrefix, p, t, info))
			}
		}

		sort.Strings(lines)
		for _, l := range lines {
			fmt.Fprint(out, l)
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

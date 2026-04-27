package debug

import (
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/ucm/direct/dresources"
	"github.com/spf13/cobra"
)

// NewRefSchemaCommand returns the hidden `ucm debug refschema` command.
// Mirrors cmd/bundle/debug/refschema.go: walks every adapter registered in
// dresources.InitAll and emits one line per (path, type) tuple with the
// sources that reference it. Output feeds the codegen pipeline that
// produces ucm/direct/dresources/{apitypes,resources}.generated.yml.
func NewRefSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refschema",
		Short: "Dump all relevant fields all ucm resources",
		Long: `Dumps all available fields for each resource type by walking the relevant types. Each line is a path to a field, type and set of tags:
- INPUT: field is present in ucm config.
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

func dumpRemoteSchemas(out io.Writer) error {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return fmt.Errorf("failed to initialize adapters: %w", err)
	}

	for _, resourceName := range slices.Sorted(maps.Keys(adapters)) {
		adapter := adapters[resourceName]

		var resourcePrefix string

		if strings.Contains(resourceName, ".") {
			resourcePrefix = "resources." + strings.ReplaceAll(resourceName, ".", ".*.")
		} else {
			resourcePrefix = "resources." + resourceName + ".*"
		}

		pathTypes := make(map[string]map[string]map[string]struct{})

		collect := func(root reflect.Type, source string) error {
			return structwalk.WalkType(root, func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
				if path == nil {
					return true
				}
				p := strings.TrimPrefix(path.String(), ".")
				if p == "permissions" || strings.HasPrefix(p, "permissions.") || strings.HasPrefix(p, "permissions[") ||
					p == "grants" || strings.HasPrefix(p, "grants.") || strings.HasPrefix(p, "grants[") {
					return false
				}
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
		for _, p := range slices.Sorted(maps.Keys(pathTypes)) {
			byType := pathTypes[p]
			for _, t := range slices.Sorted(maps.Keys(byType)) {
				info := formatTags(byType[t])
				sep := "."
				if strings.HasPrefix(p, "[") {
					sep = ""
				}
				lines = append(lines, fmt.Sprintf("%s%s%s\t%s\t%s\n", resourcePrefix, sep, p, t, info))
			}
		}

		slices.Sort(lines)
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
	return strings.Join(slices.Sorted(maps.Keys(sources)), "\t")
}

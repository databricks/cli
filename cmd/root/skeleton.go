package root

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// RegisterGenerateSkeleton adds a --generate-skeleton flag to cmd. When set, it
// prints a fillable JSON template of the command's --json request body (req must
// be a pointer to that request struct) and exits without contacting the
// workspace, so it works offline.
//
// Call it from a command override: those run after the generated command has set
// PreRunE/RunE, so this wraps both to skip the workspace-client requirement and
// the API call on the skeleton path.
func RegisterGenerateSkeleton(cmd *cobra.Command, req any) {
	var generateSkeleton bool
	cmd.Flags().BoolVar(&generateSkeleton, "generate-skeleton", false,
		`Print a fillable JSON skeleton of the --json request body and exit.`)

	// Cobra validates positional args before PreRunE/RunE, so commands that take
	// required positionals (e.g. create-endpoint NAME ENDPOINT_TYPE) would reject
	// `--generate-skeleton` with no args before we can short-circuit. Relax it on
	// the skeleton path. cmd.Args is nil for commands whose body is --json-only.
	validateArgs := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if generateSkeleton {
			return cobra.NoArgs(cmd, args)
		}
		if validateArgs == nil {
			return nil
		}
		return validateArgs(cmd, args)
	}

	mustWorkspaceClient := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if generateSkeleton {
			return nil
		}
		return mustWorkspaceClient(cmd, args)
	}

	apiCall := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !generateSkeleton {
			return apiCall(cmd, args)
		}
		if cmd.Flags().Changed("json") {
			return fmt.Errorf("--generate-skeleton cannot be combined with --json")
		}
		skeleton := jsonSkeleton(reflect.TypeOf(req).Elem(), map[reflect.Type]bool{})
		out, err := json.MarshalIndent(skeleton, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return err
	}
}

// jsonSkeleton builds a fillable example value for type t, mirroring how the SDK
// marshals the request: json tag names, pointers dereferenced, slices shown with
// a single element so nested shapes are visible. seen breaks recursive types
// (e.g. jobs.Task -> ForEachTask -> Task) so the walk terminates.
func jsonSkeleton(t reflect.Type, seen map[reflect.Type]bool) any {
	switch t.Kind() {
	case reflect.Pointer:
		return jsonSkeleton(t.Elem(), seen)
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return ""
		}
		if seen[t] {
			// Recursive type already on the current path; stop expanding it.
			return map[string]any{}
		}
		seen[t] = true
		defer delete(seen, t)
		obj := map[string]any{}
		for i := range t.NumField() {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // unexported
			}
			name, ok := jsonFieldName(f)
			if !ok {
				continue // json:"-", e.g. ForceSendFields
			}
			obj[name] = jsonSkeleton(f.Type, seen)
		}
		return obj
	case reflect.Slice, reflect.Array:
		return []any{jsonSkeleton(t.Elem(), seen)}
	case reflect.Map:
		return map[string]any{}
	case reflect.String:
		return ""
	case reflect.Bool:
		return false
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	default:
		return nil
	}
}

// jsonFieldName returns the JSON object key for a struct field and whether it is
// serialized at all (false for json:"-").
func jsonFieldName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, true
	}
	name, _, _ := strings.Cut(tag, ",")
	switch name {
	case "-":
		return "", false
	case "":
		return f.Name, true
	default:
		return name, true
	}
}

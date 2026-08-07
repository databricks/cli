package root

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/spf13/cobra"
)

// stringScalarTypes are struct types the SDK marshals to a single JSON string
// rather than an object: stdlib time.Time (RFC 3339) and the protobuf-backed
// well-known types the CLI codegen also treats as string-valued flags (see the
// IsTimestamp/IsDuration/IsFieldMask handling in internal/cligen). Reflecting
// into their unexported fields would otherwise render them as {}, so the
// skeleton emits an empty string placeholder instead. NB: every SDK request and
// message struct implements json.Marshaler too, so we cannot key off that
// interface — only these specific types marshal to a scalar.
var stringScalarTypes = map[reflect.Type]bool{
	reflect.TypeFor[time.Time]():           true,
	reflect.TypeFor[sdktime.Time]():        true,
	reflect.TypeFor[duration.Duration]():   true,
	reflect.TypeFor[fieldmask.FieldMask](): true,
}

const (
	flagSkeletonFull         = "generate-skeleton-full"
	flagSkeletonRequiredOnly = "generate-skeleton-required-only"
)

// RegisterGenerateSkeleton adds the --generate-skeleton-full and
// --generate-skeleton-required-only flags to cmd. Either prints a fillable JSON
// template of the command's --json request body (req must be a pointer to that
// request struct) and exits without contacting the workspace, so it works
// offline. --generate-skeleton-required-only keeps only the fields the API
// requires (struct fields whose json tag lacks ",omitempty"); --full keeps every
// field. Keys are sorted by name in both, so the two skeletons differ only in
// which fields are present.
//
// Call it from a command override: those run after the generated command has set
// PreRunE/RunE, so this wraps both to skip the workspace-client requirement and
// the API call on the skeleton path.
func RegisterGenerateSkeleton(cmd *cobra.Command, req any) {
	var full, requiredOnly bool
	cmd.Flags().BoolVar(&full, flagSkeletonFull, false,
		`Print a fillable JSON skeleton of the full --json request body and exit.`)
	cmd.Flags().BoolVar(&requiredOnly, flagSkeletonRequiredOnly, false,
		`Print a fillable JSON skeleton of only the required --json fields and exit.`)
	// The two skeletons are alternatives; asking for both is a user error.
	cmd.MarkFlagsMutuallyExclusive(flagSkeletonFull, flagSkeletonRequiredOnly)

	// Cobra validates positional args before PreRunE/RunE, so commands that take
	// required positionals (e.g. create-endpoint NAME ENDPOINT_TYPE) would reject
	// a skeleton flag with no args before we can short-circuit. Relax it on the
	// skeleton path. cmd.Args is nil for commands whose body is --json-only.
	validateArgs := cmd.Args
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if full || requiredOnly {
			return cobra.NoArgs(cmd, args)
		}
		if validateArgs == nil {
			return nil
		}
		return validateArgs(cmd, args)
	}

	mustWorkspaceClient := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if full || requiredOnly {
			return nil
		}
		return mustWorkspaceClient(cmd, args)
	}

	apiCall := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !full && !requiredOnly {
			return apiCall(cmd, args)
		}
		if cmd.Flags().Changed("json") {
			return errors.New("--" + skeletonFlagName(full) + " cannot be combined with --json")
		}
		skeleton := jsonSkeleton(reflect.TypeOf(req).Elem(), requiredOnly, map[reflect.Type]bool{})
		out, err := json.MarshalIndent(skeleton, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return err
	}
}

// skeletonFlagName returns the name of the skeleton flag that is set, for error
// messages.
func skeletonFlagName(full bool) string {
	if full {
		return flagSkeletonFull
	}
	return flagSkeletonRequiredOnly
}

// jsonSkeleton builds a fillable example value for type t, mirroring how the SDK
// marshals the request: json tag names, pointers dereferenced, slices shown with
// a single element so nested shapes are visible. When requiredOnly is set, only
// fields whose json tag lacks ",omitempty" are kept, matching the SDK's
// required-field convention at every nesting level. seen breaks recursive types
// (e.g. jobs.Task -> ForEachTask -> Task) so the walk terminates.
func jsonSkeleton(t reflect.Type, requiredOnly bool, seen map[reflect.Type]bool) any {
	switch t.Kind() {
	case reflect.Pointer:
		return jsonSkeleton(t.Elem(), requiredOnly, seen)
	case reflect.Struct:
		if stringScalarTypes[t] {
			return ""
		}
		if seen[t] {
			// Recursive type already on the current path; stop expanding it.
			return map[string]any{}
		}
		seen[t] = true
		defer delete(seen, t)
		obj := map[string]any{}
		for f := range t.Fields() {
			if f.PkgPath != "" {
				continue // unexported
			}
			name, optional, ok := jsonFieldName(f)
			if !ok {
				continue // json:"-", e.g. ForceSendFields
			}
			if requiredOnly && optional {
				continue
			}
			obj[name] = jsonSkeleton(f.Type, requiredOnly, seen)
		}
		return obj
	case reflect.Slice, reflect.Array:
		return []any{jsonSkeleton(t.Elem(), requiredOnly, seen)}
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

// jsonFieldName returns the JSON object key for a struct field, whether the field
// is optional (its json tag carries ",omitempty"), and whether it is serialized
// at all (false for json:"-"). The SDK marks every optional request field
// ",omitempty" and leaves required fields bare, so omitempty is the required-field
// signal at every nesting level.
func jsonFieldName(f reflect.StructField) (name string, optional, ok bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false, true
	}
	name, opts, _ := strings.Cut(tag, ",")
	optional = slices.Contains(strings.Split(opts, ","), "omitempty")
	switch name {
	case "-":
		return "", false, false
	case "":
		return f.Name, optional, true
	default:
		return name, optional, true
	}
}

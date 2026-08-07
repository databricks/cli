package root

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skeletonValue(v any, requiredOnly bool) any {
	return jsonSkeleton(reflect.TypeOf(v), requiredOnly, map[reflect.Type]bool{})
}

// skeletonJSON builds the full skeleton for the type of v and marshals it,
// mirroring what --generate-skeleton-full prints. Comparing the JSON keeps the
// assertions close to the observed behavior and avoids fighting type-aware lints
// on any values.
func skeletonJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(skeletonValue(v, false))
	require.NoError(t, err)
	return string(b)
}

func TestJSONSkeletonPrimitives(t *testing.T) {
	assert.Equal(t, `""`, skeletonJSON(t, ""))
	assert.Equal(t, `false`, skeletonJSON(t, false))
	assert.Equal(t, `0`, skeletonJSON(t, int64(0)))
	assert.Equal(t, `0`, skeletonJSON(t, float64(0)))
}

func TestJSONSkeletonPointerIsDereferenced(t *testing.T) {
	var p *string
	assert.Equal(t, `""`, skeletonJSON(t, p))
}

func TestJSONSkeletonSliceShowsOneElement(t *testing.T) {
	assert.Equal(t, `[""]`, skeletonJSON(t, []string(nil)))
}

func TestJSONSkeletonMapIsEmptyObject(t *testing.T) {
	// Maps have no fixed keys, so the shape is an empty object.
	assert.Equal(t, `{}`, skeletonJSON(t, map[string]string(nil)))
}

func TestJSONSkeletonScalarWrapperTypesAreEmptyString(t *testing.T) {
	// The SDK marshals these to a single JSON string, not an object. Reflecting
	// into their (unexported) fields would render them as {}, so the skeleton must
	// special-case them to "".
	cases := []struct {
		name string
		val  any
	}{
		{"stdlib time.Time", time.Time{}},
		{"sdk time.Time", sdktime.Time{}},
		{"sdk duration.Duration", duration.Duration{}},
		{"sdk fieldmask.FieldMask", fieldmask.FieldMask{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, `""`, skeletonJSON(t, c.val))
		})
	}
}

func TestJSONSkeletonStructUsesJSONTags(t *testing.T) {
	type inner struct {
		Value string `json:"value"`
	}
	type req struct {
		Renamed  string   `json:"renamed"`
		Untagged string   // no json tag: keeps the Go field name
		OmitTag  string   `json:"omit_tag,omitempty"`
		Nested   inner    `json:"nested"`
		Items    []inner  `json:"items"`
		Skipped  string   `json:"-"`
		unexp    string   //nolint:unused // exercises the unexported-field skip
		Force    []string `json:"-"`
	}

	want := map[string]any{
		"renamed":  "",
		"Untagged": "",
		"omit_tag": "",
		"nested":   map[string]any{"value": ""},
		"items":    []any{map[string]any{"value": ""}},
	}
	assert.Equal(t, want, skeletonValue(req{}, false))
}

func TestJSONSkeletonRequiredOnlyDropsOmitempty(t *testing.T) {
	type inner struct {
		Req string `json:"req"`
		Opt string `json:"opt,omitempty"`
	}
	type req struct {
		Required string `json:"required"`
		Optional string `json:"optional,omitempty"`
		// A required nested object keeps only its own required fields.
		Nested inner `json:"nested"`
		// An optional nested object is dropped entirely.
		OptNested inner `json:"opt_nested,omitempty"`
	}

	want := map[string]any{
		"required": "",
		"nested":   map[string]any{"req": ""},
	}
	assert.Equal(t, want, skeletonValue(req{}, true))
}

func TestJSONSkeletonRequiredOnlyRecursesThroughSlices(t *testing.T) {
	type elem struct {
		Key string `json:"key"`
		Val string `json:"val,omitempty"`
	}
	type req struct {
		Items []elem `json:"items"`
	}

	want := map[string]any{"items": []any{map[string]any{"key": ""}}}
	assert.Equal(t, want, skeletonValue(req{}, true))
}

func TestJSONFieldNameOptionalDetection(t *testing.T) {
	type s struct {
		Required string `json:"required"`
		Optional string `json:"optional,omitempty"`
		Renamed  string `json:"renamed,omitempty"`
		Skipped  string `json:"-"`
		Untagged string
	}
	byName := map[string]reflect.StructField{}
	st := reflect.TypeFor[s]()
	for f := range st.Fields() {
		byName[f.Name] = f
	}

	cases := []struct {
		field    string
		wantName string
		wantOpt  bool
		wantOK   bool
	}{
		{"Required", "required", false, true},
		{"Optional", "optional", true, true},
		{"Renamed", "renamed", true, true},
		{"Skipped", "", false, false},
		{"Untagged", "Untagged", false, true},
	}
	for _, c := range cases {
		name, opt, ok := jsonFieldName(byName[c.field])
		assert.Equal(t, c.wantName, name, c.field)
		assert.Equal(t, c.wantOpt, opt, c.field)
		assert.Equal(t, c.wantOK, ok, c.field)
	}
}

func TestJSONSkeletonRecursiveTypeTerminates(t *testing.T) {
	// A self-referential type would recurse forever without the seen guard; the
	// back-edge collapses to an empty object.
	type node struct {
		Name  string  `json:"name"`
		Child *node   `json:"child"`
		Kids  []*node `json:"kids"`
	}

	want := map[string]any{
		"name":  "",
		"child": map[string]any{},
		"kids":  []any{map[string]any{}},
	}
	assert.Equal(t, want, skeletonValue(node{}, false))
}

// requestBody is a stand-in request struct for the flag-wiring tests.
type requestBody struct {
	Name  string `json:"name"`
	Extra string `json:"extra,omitempty"`
}

func newSkeletonTestCmd(req *requestBody) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create NAME",
		Args: cobra.ExactArgs(1),
		// Mimic a generated command: --json unmarshals into the request, and the
		// real work needs a workspace client.
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return assert.AnError
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return assert.AnError
		},
	}
	cmd.Flags().String("json", "", "request body")
	RegisterGenerateSkeleton(cmd, req)
	return cmd
}

func runSkeletonCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newSkeletonTestCmd(&requestBody{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestRegisterGenerateSkeletonFullOffline(t *testing.T) {
	// A skeleton flag must not require positional args or a workspace client.
	out, err := runSkeletonCmd(t, "--"+flagSkeletonFull)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, map[string]any{"name": "", "extra": ""}, parsed)
}

func TestRegisterGenerateSkeletonRequiredOnly(t *testing.T) {
	out, err := runSkeletonCmd(t, "--"+flagSkeletonRequiredOnly)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	// Only the required field survives; "extra" is omitempty.
	assert.Equal(t, map[string]any{"name": ""}, parsed)
}

func TestRegisterGenerateSkeletonFlagsMutuallyExclusive(t *testing.T) {
	_, err := runSkeletonCmd(t, "--"+flagSkeletonFull, "--"+flagSkeletonRequiredOnly)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "none of the others can be")
}

func TestRegisterGenerateSkeletonRejectsJSON(t *testing.T) {
	_, err := runSkeletonCmd(t, "--"+flagSkeletonRequiredOnly, "--json", "{}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined with --json")
}

func TestRegisterGenerateSkeletonPreservesNormalPath(t *testing.T) {
	// Without a skeleton flag, the original PreRunE runs (returns before the API
	// call).
	_, err := runSkeletonCmd(t, "some-name")
	assert.ErrorIs(t, err, assert.AnError)
}

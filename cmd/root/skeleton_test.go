package root

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skeletonValue(v any) any {
	return jsonSkeleton(reflect.TypeOf(v), map[reflect.Type]bool{})
}

// skeletonJSON builds the skeleton for the type of v and marshals it, mirroring
// what --generate-skeleton prints. Comparing the JSON keeps the assertions close
// to the observed behavior and avoids fighting type-aware lints on any values.
func skeletonJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(skeletonValue(v))
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

func TestJSONSkeletonTimeIsEmptyString(t *testing.T) {
	// The SDK marshals time.Time as an RFC 3339 string, not a struct.
	assert.Equal(t, `""`, skeletonJSON(t, time.Time{}))
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
	assert.Equal(t, want, skeletonValue(req{}))
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
	assert.Equal(t, want, skeletonValue(node{}))
}

// requestBody is a stand-in request struct for the flag-wiring tests.
type requestBody struct {
	Name string `json:"name"`
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

func TestRegisterGenerateSkeletonPrintsSkeletonOffline(t *testing.T) {
	// --generate-skeleton must not require positional args or a workspace client.
	out, err := runSkeletonCmd(t, "--generate-skeleton")
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, map[string]any{"name": ""}, parsed)
}

func TestRegisterGenerateSkeletonRejectsJSON(t *testing.T) {
	_, err := runSkeletonCmd(t, "--generate-skeleton", "--json", "{}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined with --json")
}

func TestRegisterGenerateSkeletonPreservesNormalPath(t *testing.T) {
	// Without the flag, the original PreRunE runs (returns before the API call).
	_, err := runSkeletonCmd(t, "some-name")
	assert.ErrorIs(t, err, assert.AnError)
}

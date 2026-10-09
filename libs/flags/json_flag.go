package flags

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/marshal"
	"go.yaml.in/yaml/v3"
)

type JsonFlag struct {
	raw    []byte
	source string
}

func (j *JsonFlag) String() string {
	return fmt.Sprintf("JSON (%d bytes)", len(j.raw))
}

// TODO: Command.MarkFlagFilename()
func (j *JsonFlag) Set(v string) error {
	if v == "" {
		return errors.New("expected inline JSON or @path/to/file, got an empty string")
	}
	// Load request from file if it starts with '@' (like curl).
	if v[0] != '@' {
		j.raw = []byte(v)
		j.source = "(inline)"
		return nil
	}
	filePath := v[1:]
	buf, err := os.ReadFile(filePath)
	j.source = filePath
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}
	j.raw = buf
	return nil
}

func (j *JsonFlag) Unmarshal(v any) diag.Diagnostics {
	if j.raw == nil {
		return nil
	}

	node, err := structvar.ParseJSON(j.source, j.raw)
	if err != nil {
		return diag.FromErr(err)
	}

	// Convert the input to the types of the target, in a fresh value: fields set by
	// other flags are already in v and must be kept.
	// For example string literals for booleans and integers will be converted to the correct types.
	sv, diags, err := structvar.DecodeYAMLNode(j.source, node, reflect.New(reflect.TypeOf(v).Elem()).Interface(), nil)
	if err != nil {
		var expectedJsonType string
		switch reflect.TypeOf(v).Elem().Kind() {
		case reflect.Struct, reflect.Map:
			expectedJsonType = "object"
		case reflect.Slice:
			expectedJsonType = "array"
		default:
			expectedJsonType = "string"
		}

		return diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid command input",
			Detail:   fmt.Sprintf("expected JSON %s, received %s", expectedJsonType, nodeKind(node)),
		})
	}

	// Then marshal the normalized data to the output.
	// It will serialize all set data with the correct types.
	data, err := json.Marshal(sv.View().AsAny())
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	if reflect.TypeOf(v).Elem().Kind() == reflect.Struct {
		// Finally unmarshal the normalized data to the output.
		// It will fill in the ForceSendFields field if the struct contains it.
		err = marshal.Unmarshal(data, v)
	} else {
		// If the output is not a struct, just unmarshal the data to the output.
		err = json.Unmarshal(data, v)
	}
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}
	return diags
}

func (j *JsonFlag) Type() string {
	return "JSON"
}

// Raw returns the raw JSON bytes the flag was set to (or nil when the flag
// was not provided). Exposed so command overrides can do shape-level
// validation before the generated Unmarshal call.
func (j *JsonFlag) Raw() []byte {
	return j.raw
}

// Validate parses the JSON for well-formedness and returns the same positioned
// error Unmarshal would (including the @file source in the location), without
// decoding it into a value. Callers that send the raw bytes verbatim use this
// to keep client-side validation. Returns nil when the flag was not set.
func (j *JsonFlag) Validate() error {
	if j.raw == nil {
		return nil
	}
	_, err := structvar.ParseJSON(j.source, j.raw)
	return err
}

// RejectWrappedJSON returns a clear client-side error when the --json body
// is a top-level object containing outerKey. It detects the common mistake
// of wrapping the body in '{"<outerKey>": ...}' when the flag binds to the
// inner object (the SDK request type's JSON-tagged outer field). Returns
// nil when the flag is unset, when the body isn't an object, or when no
// outerKey is present at the top level.
//
// example, if non-empty, is appended to the error as a hint at the correct
// shape. Callers typically construct it as a self-contained command line.
//
// This is the shared helper used by the postgres command overrides; the
// same inner-body --json shape exists across siblings like create-branch,
// create-database, create-endpoint, and create-project.
func (j *JsonFlag) RejectWrappedJSON(outerKey, example string) error {
	if len(j.raw) == 0 {
		return nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(j.raw, &top); err != nil {
		// Defer non-object inputs to the generated unmarshal so its
		// diagnostics render the original parse error.
		return nil //nolint:nilerr
	}
	if _, found := top[outerKey]; !found {
		return nil
	}
	msg := fmt.Sprintf("--json should NOT be wrapped in '{%q: ...}'.\n\n"+
		"The flag binds to the inner object — supply its fields directly.", outerKey)
	if example != "" {
		msg += "\n\nExample:\n\n  " + example
	}
	return fmt.Errorf("%s", msg)
}

// nodeKind returns the kind of the JSON value in node, as a user sees it.
func nodeKind(node *yaml.Node) string {
	switch node.Kind {
	case yaml.MappingNode:
		return "map"
	case yaml.SequenceNode:
		return "sequence"
	default:
		return map[string]string{"!!str": "string", "!!bool": "bool", "!!int": "int", "!!float": "float", "!!null": "nil"}[node.Tag]
	}
}

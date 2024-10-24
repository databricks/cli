package flags

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/databricks-sdk-go/marshal"
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

	dv, err := jsonloader.LoadJSON(j.raw, j.source)
	if err != nil {
		return diag.FromErr(err)
	}

	// First normalize the input data.
	// It will convert all the values to the correct types.
	// For example string literals for booleans and integers will be converted to the correct types.
	nv, diags := convert.Normalize(v, dv)
	if diags.HasError() {
		return diags
	}

	// Then marshal the normalized data to the output.
	// It will serialize all set data with the correct types.
	data, err := json.Marshal(nv.AsAny())
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	kind := reflect.ValueOf(v).Kind()
	if kind == reflect.Ptr {
		kind = reflect.ValueOf(v).Elem().Kind()
	}

	if kind == reflect.Struct {
		// Finally unmarshal the normalized data to the output.
		// It will fill in the ForceSendFields field if the struct contains it.
		err = marshal.Unmarshal(data, v)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
	} else {
		// If the output is not a struct, just unmarshal the data to the output.
		err = json.Unmarshal(data, v)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
	}

	return diags
}

func (j *JsonFlag) Type() string {
	return "JSON"
}

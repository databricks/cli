package bundle

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestTelemetryBoolValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		tm   Telemetry
		want []protos.BoolMapEntry
	}{
		{"zero value reports nothing", Telemetry{}, nil},
		{"bool false is not reported", Telemetry{SelectUsed: false}, nil},
		{"bool true", Telemetry{SelectUsed: true}, []protos.BoolMapEntry{
			{Key: "select_used", Value: true},
		}},
		{"*bool nil is not reported", Telemetry{RunAsSet: nil}, nil},
		{"*bool false", Telemetry{RunAsSet: new(false)}, []protos.BoolMapEntry{
			{Key: "run_as_set", Value: false},
		}},
		{"*bool true", Telemetry{RunAsSet: new(true)}, []protos.BoolMapEntry{
			{Key: "run_as_set", Value: true},
		}},
		{"[]bool empty is not reported", Telemetry{ArtifactFilesIsSet: []bool{}}, nil},
		{"[]bool reports every value in order", Telemetry{ArtifactFilesIsSet: []bool{true, false, true}}, []protos.BoolMapEntry{
			{Key: "artifact_files_is_set", Value: true},
			{Key: "artifact_files_is_set", Value: false},
			{Key: "artifact_files_is_set", Value: true},
		}},
		{"fields are reported in declaration order", Telemetry{RunAsSet: new(true), SelectUsed: true}, []protos.BoolMapEntry{
			{Key: "select_used", Value: true},
			{Key: "run_as_set", Value: true},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.tm.BoolValues())
		})
	}
}

// BoolValues and the bitmap skip fields of any other type, and an empty or
// duplicate key would be indistinguishable on the wire.
func TestTelemetryFields(t *testing.T) {
	supported := []reflect.Type{reflect.TypeFor[bool](), reflect.TypeFor[*bool](), reflect.TypeFor[[]bool]()}
	keys := map[string]string{}
	for _, f := range reflect.VisibleFields(reflect.TypeFor[Telemetry]()) {
		assert.Contains(t, supported, f.Type, "field %s", f.Name)
		key := f.Tag.Get("metric")
		assert.NotEmpty(t, key, "field %s has no metric tag", f.Name)
		if other, ok := keys[key]; ok {
			assert.Fail(t, "duplicate metric key", "%q is used by %s and %s", key, other, f.Name)
		}
		keys[key] = f.Name
	}
}

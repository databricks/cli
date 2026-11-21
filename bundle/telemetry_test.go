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
		{
			name: "zero value reports nothing",
			tm:   Telemetry{},
			want: nil,
		},
		{
			name: "bool false is not reported",
			tm:   Telemetry{SelectUsed: false},
			want: nil,
		},
		{
			name: "bool true",
			tm:   Telemetry{SelectUsed: true},
			want: []protos.BoolMapEntry{{Key: "select_used", Value: true}},
		},
		{
			name: "*bool nil is not reported",
			tm:   Telemetry{RunAsSet: nil},
			want: nil,
		},
		{
			name: "*bool false",
			tm:   Telemetry{RunAsSet: new(false)},
			want: []protos.BoolMapEntry{{Key: "run_as_set", Value: false}},
		},
		{
			name: "*bool true",
			tm:   Telemetry{RunAsSet: new(true)},
			want: []protos.BoolMapEntry{{Key: "run_as_set", Value: true}},
		},
		{
			name: "BoolSet unset is not reported",
			tm:   Telemetry{ArtifactFilesIsSet: boolSet()},
			want: nil,
		},
		{
			name: "BoolSet reports each value once, true first",
			tm:   Telemetry{ArtifactFilesIsSet: boolSet(false, true, false)},
			want: []protos.BoolMapEntry{
				{Key: "artifact_files_is_set", Value: true},
				{Key: "artifact_files_is_set", Value: false},
			},
		},
		{
			name: "fields are reported in declaration order",
			tm:   Telemetry{RunAsSet: new(true), SelectUsed: true},
			want: []protos.BoolMapEntry{
				{Key: "select_used", Value: true},
				{Key: "run_as_set", Value: true},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.tm.BoolValues())
		})
	}
}

// BoolValues and the bitmap skip fields of any other type, and an empty or
// duplicate key would be indistinguishable on the wire.
func TestTelemetryFields(t *testing.T) {
	supported := []reflect.Type{reflect.TypeFor[bool](), reflect.TypeFor[*bool](), reflect.TypeFor[BoolSet]()}
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

func TestBoolSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  []bool
		want []bool
	}{
		{
			name: "never set",
			set:  nil,
			want: nil,
		},
		{
			name: "true",
			set:  []bool{true, true},
			want: []bool{true},
		},
		{
			name: "false",
			set:  []bool{false},
			want: []bool{false},
		},
		{
			name: "both, true first",
			set:  []bool{false, true},
			want: []bool{true, false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, boolSet(tc.set...).Values())
		})
	}
}

func boolSet(values ...bool) BoolSet {
	var s BoolSet
	for _, v := range values {
		s.SetBool(v)
	}
	return s
}

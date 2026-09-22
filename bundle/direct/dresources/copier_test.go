package dresources

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/stretchr/testify/assert"
)

// TestNoRedundantRemapState enforces the "only override when needed" rule: a resource that
// keeps a hand-written RemapState must actually do something the auto-generated copier cannot.
// For every resource that still has a RemapState whose remote and state types differ, it fully
// populates a remote value and fails if the copier would produce an identical result — meaning
// the method is dead boilerplate that should be deleted (see buildCopiers).
func TestNoRedundantRemapState(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var redundant []string

	for resourceType, resource := range SupportedResources {
		implType := reflect.TypeOf(resource)
		if seen[implType] {
			continue
		}
		seen[implType] = true

		m := reflect.ValueOf(resource).MethodByName("RemapState")
		if !m.IsValid() {
			continue
		}
		remoteType := m.Type().In(0)
		stateType := m.Type().Out(0)
		if remoteType == stateType {
			continue // identity, the copier is not involved
		}

		copier, err := structcopy.Compile(remoteType, stateType)
		if err != nil {
			continue // copier cannot handle it, so the override is required
		}

		remote := reflect.New(remoteType.Elem())
		fillNonZero(remote.Elem(), 0)
		setForceSendFields(remote.Elem())

		want := m.Call([]reflect.Value{remote})[0].Interface()
		got := copier.Copy(remote.Interface())
		if reflect.DeepEqual(want, got) {
			redundant = append(redundant, resourceType)
		}
	}

	slices.Sort(redundant)
	assert.Empty(t, redundant, "these resources have a RemapState the auto-copier reproduces exactly; delete the method and let buildCopiers handle it")
}

// setForceSendFields recursively sets every struct's ForceSendFields to its own field names,
// so the copier's ForceSendFields-filtering path is exercised at every level. fillNonZero
// deliberately leaves ForceSendFields empty (round-trip tests need that), so we populate it
// here. Map values are not addressable and cannot hold a relevant FSF source, so they are skipped.
func setForceSendFields(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			setForceSendFields(v.Elem())
		}
	case reflect.Slice:
		for i := range v.Len() {
			setForceSendFields(v.Index(i))
		}
	case reflect.Struct:
		var names []string
		for i := range v.NumField() {
			sf := v.Type().Field(i)
			if !sf.IsExported() || sf.Name == "ForceSendFields" {
				continue
			}
			names = append(names, sf.Name)
			setForceSendFields(v.Field(i))
		}
		if f := v.FieldByName("ForceSendFields"); f.IsValid() && f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
			f.Set(reflect.ValueOf(names))
		}
	default:
		// scalars and other kinds have no nested ForceSendFields to set
	}
}

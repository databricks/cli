package dresources

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertFieldsCovered asserts that all fields in sdkType (except those in skip)
// are present as direct fields in remoteType, and that skipped fields are indeed absent.
func assertFieldsCovered(t *testing.T, sdkType, remoteType reflect.Type, skip map[string]bool) {
	t.Helper()
	remoteFields := map[string]bool{}
	for i := range remoteType.NumField() {
		f := remoteType.Field(i)
		if !f.Anonymous {
			remoteFields[f.Name] = true
		}
	}

	for i := range sdkType.NumField() {
		field := sdkType.Field(i)
		if skip[field.Name] {
			assert.NotContains(t, remoteFields, field.Name, "field %s is in skip list but present in %s; remove it from skip", field.Name, remoteType.Name())
			continue
		}
		assert.Contains(t, remoteFields, field.Name, "field %s from %s is missing in %s", field.Name, sdkType.Name(), remoteType.Name())
	}
}

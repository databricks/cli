package root_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Every spec field must reach the command, and changing it while building
// must panic. This fails if a field is added to Spec but not handled.
func TestSpecFields(t *testing.T) {
	specType := reflect.TypeFor[root.CommandSpec]()
	for i := range specType.NumField() {
		field := specType.Field(i)
		if field.Name == "Build" {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			spec := root.CommandSpec{Use: "cmd"}
			set := func(v reflect.Value) {
				switch v.Kind() {
				case reflect.String:
					v.SetString(v.String() + "x")
				case reflect.Bool:
					v.SetBool(!v.Bool())
				case reflect.Slice:
					v.Set(reflect.Append(v, reflect.ValueOf("x")))
				default:
					t.Fatalf("unhandled kind %s", v.Kind())
				}
			}
			set(reflect.ValueOf(&spec).Elem().Field(i))

			spec.Build = func(context.Context, *cobra.Command) {}
			cmd := root.NewCommand(t.Context(), spec)
			assert.Equal(t, reflect.ValueOf(spec).Field(i).Interface(), reflect.ValueOf(cmd).Elem().FieldByName(field.Name).Interface())

			spec.Build = func(_ context.Context, cmd *cobra.Command) { set(reflect.ValueOf(cmd).Elem().FieldByName(field.Name)) }
			assert.Panics(t, func() { root.NewCommand(t.Context(), spec) })
		})
	}
}

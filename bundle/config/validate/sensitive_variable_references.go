package validate

import (
	"context"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

type sensitiveVariableReferences struct{}

// SensitiveVariableReferences validates that a variable marked `sensitive: true` is only
// referenced from places that are themselves sensitive: a resource field listed in the
// resource type's sensitive_fields, or another sensitive variable. Anywhere else the
// secret would end up in plain text in the deployed resource or in CLI output.
//
// It inspects unresolved references, so it must run before variable resolution.
func SensitiveVariableReferences() bundle.Mutator {
	return &sensitiveVariableReferences{}
}

func (m *sensitiveVariableReferences) Name() string {
	return "validate:sensitive_variable_references"
}

func (m *sensitiveVariableReferences) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !variable.AnySensitive(b.Config.Variables) {
		return nil
	}

	var diags diag.Diagnostics
	root := b.Config.Value()
	_ = dyn.WalkReadOnly(root, func(p dyn.Path, v dyn.Value) error {
		// Target overrides are merged into the root before this runs; walking them
		// too would report the same reference twice and flag targets not being deployed.
		if len(p) == 1 && p[0].Key() == "targets" {
			return dyn.ErrSkip
		}
		// SetVariables copies the default into the value; skip the copy so that a
		// reference in the default is reported once.
		if len(p) == 3 && p[0].Key() == "variables" && p[2].Key() == "value" {
			d, err := dyn.GetByPath(root, dyn.NewPath(p[0], p[1], dyn.Key("default")))
			if err == nil && reflect.DeepEqual(d.AsAny(), v.AsAny()) {
				return dyn.ErrSkip
			}
		}
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			name, ok := referencedVariable(r)
			if !ok || !isSensitiveVariable(b, name) || isSensitiveLocation(b, p) {
				continue
			}
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Sensitive variable %q is used in a non-sensitive field", name),
				Detail: fmt.Sprintf(`The variable "%s" is marked as sensitive, so its value must not appear in plain text.
Sensitive variables can only be used as the value of a sensitive resource field (such as resources.secrets.*.value) or in the definition of another variable that is also marked as sensitive.`, name),
				Locations: v.Locations(),
				Paths:     dyn.ToStructPaths(p),
			})
		}
		return nil
	})

	return diags
}

func isSensitiveVariable(b *bundle.Bundle, name string) bool {
	v, ok := b.Config.Variables[name]
	return ok && v.Sensitive
}

// referencedVariable returns the variable a reference like "var.foo" or
// "variables.foo.value" points to.
func referencedVariable(ref string) (string, bool) {
	p, err := dyn.NewPathFromString(ref)
	if err != nil || len(p) < 2 {
		return "", false
	}
	switch p[0].Key() {
	case "var", "variables":
		return p[1].Key(), true
	default:
		return "", false
	}
}

// isSensitiveLocation reports whether a value at path p may hold a secret: it is part
// of the definition of a sensitive variable, or a field under resources.<type>.<key>
// that the resource type declares as sensitive.
func isSensitiveLocation(b *bundle.Bundle, p dyn.Path) bool {
	if len(p) >= 2 && p[0].Key() == "variables" {
		return isSensitiveVariable(b, p[1].Key())
	}
	if len(p) < 4 || p[0].Key() != "resources" {
		return false
	}
	field, err := structpath.ParsePath(p[3:].String())
	if err != nil {
		return false
	}
	for _, rule := range dresources.GetResourceConfig(p[1].Key()).SensitiveFields {
		if field.HasPatternPrefix(rule.Field) {
			return true
		}
	}
	return false
}

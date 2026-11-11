package validate

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

type sensitiveVariableReferences struct{}

// SensitiveVariableReferences marks every variable referenced from a sensitive resource
// field (a field listed in the resource type's sensitive_fields, such as a secret's value)
// as sensitive, then validates that a sensitive variable is only referenced from places
// that are themselves sensitive: a sensitive resource field or another sensitive variable.
// Anywhere else the secret would end up in plain text in the deployed resource or in CLI
// output.
//
// It inspects unresolved references, so it must run before variable resolution.
func SensitiveVariableReferences() bundle.Mutator {
	return &sensitiveVariableReferences{}
}

func (m *sensitiveVariableReferences) Name() string {
	return "validate:sensitive_variable_references"
}

// variableReference is an unresolved reference to the variable name, found in value at path.
type variableReference struct {
	path  dyn.Path
	name  string
	value dyn.Value
}

func (m *sensitiveVariableReferences) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	refs := variableReferences(b.Config.Value())

	err := inferSensitiveVariables(b, refs)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, r := range refs {
		if !isSensitiveVariable(b, r.name) || isSensitiveLocation(b, r.path) {
			continue
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Sensitive variable %q is used in a non-sensitive field", r.name),
			Detail: fmt.Sprintf(`The variable "%s" is sensitive, so its value must not appear in plain text.
A variable is sensitive when it is marked with "sensitive: true" or used as the value of a sensitive resource field (such as resources.secrets.*.value). It can only be used in such a field or in the definition of another sensitive variable.`, r.name),
			Locations: r.value.Locations(),
			Paths:     dyn.ToStructPaths(r.path),
		})
	}
	return diags
}

// variableReferences collects the unresolved variable references in the config.
func variableReferences(root dyn.Value) []variableReference {
	var refs []variableReference
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
			if !ok {
				continue
			}
			// The walk reuses the path's backing array for sibling nodes.
			refs = append(refs, variableReference{path: slices.Clone(p), name: name, value: v})
		}
		return nil
	})
	return refs
}

// inferSensitiveVariables marks every variable referenced from a sensitive resource
// field as sensitive, so the marker does not have to be repeated on such a variable.
func inferSensitiveVariables(b *bundle.Bundle, refs []variableReference) error {
	return b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, r := range refs {
			// A reference to an undeclared variable is reported by variable resolution.
			v, ok := b.Config.Variables[r.name]
			if !ok || config.IsExplicitlyEnabled(v.Sensitive) || !isSensitiveResourceField(r.path) {
				continue
			}
			var err error
			root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("variables"), dyn.Key(r.name), dyn.Key("sensitive")), dyn.V(true))
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return root, nil
	})
}

func isSensitiveVariable(b *bundle.Bundle, name string) bool {
	v, ok := b.Config.Variables[name]
	return ok && config.IsExplicitlyEnabled(v.Sensitive)
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
// of the definition of a sensitive variable or a sensitive resource field.
func isSensitiveLocation(b *bundle.Bundle, p dyn.Path) bool {
	if len(p) >= 2 && p[0].Key() == "variables" {
		return isSensitiveVariable(b, p[1].Key())
	}
	return isSensitiveResourceField(p)
}

// isSensitiveResourceField reports whether p is a field under resources.<type>.<key>
// that the resource type declares as sensitive.
func isSensitiveResourceField(p dyn.Path) bool {
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

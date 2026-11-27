package validate

import (
	"context"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
	path  *structpath.PathNode
	name  string
	value structvar.View
}

func (m *sensitiveVariableReferences) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	refs := variableReferences(b.Config.View())

	inferSensitiveVariables(b, refs)

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
			Paths:     []*structpath.PathNode{r.path},
		})
	}
	return diags
}

// variableReferences collects the unresolved variable references in the config.
func variableReferences(root structvar.View) []variableReference {
	var refs []variableReference
	_ = structvar.Walk(root, func(p *structpath.PathNode, v structvar.View) error {
		// Target overrides are merged into the root before this runs; walking them
		// too would report the same reference twice and flag targets not being deployed.
		if p.Len() == 1 && p.KeyAt(0) == "targets" {
			return structvar.ErrSkip
		}
		// SetVariables copies the default into the value; skip the copy so that a
		// reference in the default is reported once.
		if p.Len() == 3 && p.KeyAt(0) == "variables" && p.KeyAt(2) == "value" {
			d := root.Get("variables").Get(p.KeyAt(1)).Get("default")
			if d.IsValid() && reflect.DeepEqual(d.AsAny(), v.AsAny()) {
				return structvar.ErrSkip
			}
		}
		s, ok := v.AsString()
		if !ok {
			return nil
		}
		ref, ok := structvar.NewRef(s)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			name, ok := referencedVariable(r)
			if !ok {
				continue
			}
			refs = append(refs, variableReference{path: p, name: name, value: v})
		}
		return nil
	})
	return refs
}

// inferSensitiveVariables marks every variable referenced from a sensitive resource
// field as sensitive, so the marker does not have to be repeated on such a variable.
func inferSensitiveVariables(b *bundle.Bundle, refs []variableReference) {
	for _, r := range refs {
		// A reference to an undeclared variable is reported by variable resolution.
		v, ok := b.Config.Variables[r.name]
		if !ok || v == nil || config.IsExplicitlyEnabled(v.Sensitive) || !isSensitiveResourceField(r.path) {
			continue
		}
		v.Sensitive = new(true)
	}
}

func isSensitiveVariable(b *bundle.Bundle, name string) bool {
	v, ok := b.Config.Variables[name]
	return ok && v != nil && config.IsExplicitlyEnabled(v.Sensitive)
}

// referencedVariable returns the variable a reference like "var.foo" or
// "variables.foo.value" points to.
func referencedVariable(ref string) (string, bool) {
	p, err := structpath.ParsePath(ref)
	if err != nil || p.Len() < 2 {
		return "", false
	}
	switch p.KeyAt(0) {
	case "var", "variables":
		return p.KeyAt(1), p.KeyAt(1) != ""
	default:
		return "", false
	}
}

// isSensitiveLocation reports whether a value at path p may hold a secret: it is part
// of the definition of a sensitive variable or a sensitive resource field.
func isSensitiveLocation(b *bundle.Bundle, p *structpath.PathNode) bool {
	if p.Len() >= 2 && p.KeyAt(0) == "variables" {
		return isSensitiveVariable(b, p.KeyAt(1))
	}
	return isSensitiveResourceField(p)
}

// isSensitiveResourceField reports whether p is a field under resources.<type>.<key>
// that the resource type declares as sensitive.
func isSensitiveResourceField(p *structpath.PathNode) bool {
	if p.Len() < 4 || p.KeyAt(0) != "resources" {
		return false
	}
	field := p.SkipPrefix(3)
	for _, rule := range dresources.GetResourceConfig(p.KeyAt(1)).SensitiveFields {
		if field.HasPatternPrefix(rule.Field) {
			return true
		}
	}
	return false
}

package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type validateSecretValueIsVariable struct{}

func ValidateSecretValueIsVariable() bundle.Mutator {
	return &validateSecretValueIsVariable{}
}

func (v *validateSecretValueIsVariable) Name() string {
	return "ValidateSecretValueIsVariable"
}

func (v *validateSecretValueIsVariable) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Iterate over all secrets in the bundle
	for key := range b.Config.Resources.Secrets {
		p := structpath.NewPath(nil, "resources", "secrets", key, "value")
		val := b.Config.View().Lookup(p)
		if !val.IsValid() {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Secret value must be a string",
				Detail:   fmt.Sprintf(`The secret value for "%s" must be a string.`, key),
				Paths:    []*structpath.PathNode{p},
			})
			continue
		}

		valueStr, ok := val.AsString()
		if !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Secret value must be a string",
				Detail:   fmt.Sprintf(`The secret value for "%s" must be a string.`, key),
				Paths:    []*structpath.PathNode{p},
			})
			continue
		}

		// Value must be a variable reference to prevent leaking secrets in config files
		if !structvar.IsPureVariableReference(valueStr) {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Secret value must be a variable reference",
				Detail: fmt.Sprintf(`The secret value for "%s" must be a variable reference (e.g., ${var.my_secret}).
Plain text secret values are not allowed to prevent leaking secrets in configuration files.
Use bundle variables to pass secret values at deployment time.`, key),
				Paths: []*structpath.PathNode{p},
			})
			continue
		}

		// The value is a variable reference. If it is a pure ${var.<name>} reference,
		// check that the referenced variable does not have a default value set. A default
		// value would be stored in plain text in the config file, defeating the purpose
		// of using a variable reference.
		diags = append(diags, v.checkVariableDefault(b, key, valueStr, p, val)...)
	}

	return diags
}

// checkVariableDefault emits an error if valueStr is a pure ${var.<name>} reference
// and the referenced variable has a default value set.
func (v *validateSecretValueIsVariable) checkVariableDefault(b *bundle.Bundle, secretKey, valueStr string, p *structpath.PathNode, val structvar.View) diag.Diagnostics {
	refPath, ok := structvar.PureReferenceToPath(valueStr)
	if !ok || refPath.Len() < 2 || refPath.KeyAt(0) != "var" {
		return nil
	}

	varName := refPath.KeyAt(1)
	variable, exists := b.Config.Variables[varName]
	if !exists || variable == nil || !variable.HasDefault() {
		return nil
	}

	// The default path in the dynamic config for the variable's default field.
	defaultPath := structpath.NewPath(nil, "variables", varName, "default")
	locations := append(b.Config.LocationsAt(defaultPath), val.Locations()...)

	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "Variable used for secret value must not have a default value",
		Detail: fmt.Sprintf(`The variable "%s" used for the secret "%s" must not have a default value.
A default value is stored in plain text in the configuration file, which defeats the purpose of using a variable reference for a secret.
Remove the default value and pass the secret value at deployment time using "--var", the BUNDLE_VAR_%s environment variable, or a variable overrides file.`, varName, secretKey, varName),
		Locations: locations,
		Paths:     []*structpath.PathNode{p},
	}}
}

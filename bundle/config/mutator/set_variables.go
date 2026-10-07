package mutator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

const bundleVarPrefix = "BUNDLE_VAR_"

type setVariables struct{}

func SetVariables() bundle.Mutator {
	return &setVariables{}
}

func (m *setVariables) Name() string {
	return "SetVariables"
}

func getDefaultVariableFilePath(target string) string {
	return ".databricks/bundle/" + target + "/variable-overrides.json"
}

func setVariable(ctx context.Context, cfg *config.Root, variable *variable.Variable, name string, fileDefault structvar.View) error {
	// case: variable already has value initialized, so skip
	if variable.HasValue() {
		return nil
	}

	variablePath := structpath.NewStringKeys(nil, "variables", name)
	valuePath := structpath.NewStringKey(variablePath, "value")

	// case: read and set variable value from process environment
	envVarName := bundleVarPrefix + name
	if val, ok := env.Lookup(ctx, envVarName); ok {
		if variable.IsComplex() {
			return fmt.Errorf(`setting via environment variables (%s) is not supported for complex variable %s`, envVarName, name)
		}

		err := cfg.Set(valuePath, val)
		if err != nil {
			return fmt.Errorf(`failed to assign value "%s" to variable %s from environment variable %s with error: %v`, val, name, envVarName, err)
		}
		return nil
	}

	// case: Defined a variable for named lookup for a resource
	// It will be resolved later in ResolveResourceReferences mutator
	if variable.Lookup != nil {
		return nil
	}

	// case: Set the variable to the default value from the variable file
	if fileDefault.Kind() != structvar.KindInvalid && fileDefault.Kind() != structvar.KindNil {
		hasComplexType := variable.IsComplex()
		hasComplexValue := fileDefault.Kind() == structvar.KindMap || fileDefault.Kind() == structvar.KindSequence

		if hasComplexType && !hasComplexValue {
			return fmt.Errorf(`variable %s is of type complex, but the value in the variable file is not a complex type`, name)
		}
		if !hasComplexType && hasComplexValue {
			return fmt.Errorf(`variable %s is not of type complex, but the value in the variable file is a complex type`, name)
		}

		err := cfg.Assign(valuePath, fileDefault)
		if err != nil {
			return fmt.Errorf(`failed to assign default value from variable file to variable %s with error: %v`, name, err)
		}

		return nil
	}

	// case: Set the variable to its default value
	if variable.HasDefault() {
		vDefault := cfg.View().Lookup(structpath.NewStringKey(variablePath, "default"))
		if !vDefault.IsValid() {
			return fmt.Errorf(`failed to get default value from config "%s" for variable %s with error: no such key: default`, variable.Default, name)
		}

		err := cfg.Assign(valuePath, vDefault)
		if err != nil {
			return fmt.Errorf(`failed to assign default value from config "%s" to variable %s with error: %v`, variable.Default, name, err)
		}
		return nil
	}

	// We should have had a value to set for the variable at this point.
	return fmt.Errorf(`no value assigned to required variable %s. Variables are usually assigned in databricks.yml, and they can be overridden using "--var", the %s environment variable, or %s`, name, bundleVarPrefix+name, getDefaultVariableFilePath("<target>"))
}

func readVariablesFromFile(b *bundle.Bundle) (structvar.View, diag.Diagnostics) {
	var diags diag.Diagnostics

	filePath := filepath.Join(b.BundleRootPath, getDefaultVariableFilePath(b.Config.Bundle.Target))
	if _, err := os.Stat(filePath); err != nil {
		return structvar.View{}, nil
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return structvar.View{}, diag.FromErr(fmt.Errorf("failed to read variables file: %w", err))
	}

	node, err := structvar.ParseJSON(filePath, f)
	if err != nil {
		return structvar.View{}, diag.FromErr(fmt.Errorf("failed to parse variables file %s: %w", filePath, err))
	}
	var v any
	decoded, _, err := structvar.DecodeYAMLNode(filePath, node, &v, nil)
	if err != nil {
		return structvar.View{}, diag.FromErr(fmt.Errorf("failed to parse variables file %s: %w", filePath, err))
	}

	val := decoded.View()
	if val.Kind() != structvar.KindMap {
		return structvar.View{}, diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to parse variables file %s: invalid format", filePath),
			Detail:   "Variables file must be a JSON object with the following format:\n{\"var1\": \"value1\", \"var2\": \"value2\"}",
		})
	}

	return val, nil
}

func (m *setVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	defaults, diags := readVariablesFromFile(b)
	if diags.HasError() {
		return diags
	}
	for name := range b.Config.View().Get("variables").MapItems() {
		v, ok := b.Config.Variables[name]
		if !ok {
			return diags.Extend(diag.Errorf(`variable "%s" is not defined`, name))
		}

		fileDefault := defaults.Get(name)
		err := setVariable(ctx, &b.Config, v, name, fileDefault)
		if err != nil {
			return diags.Extend(diag.FromErr(err))
		}
	}

	return diags
}

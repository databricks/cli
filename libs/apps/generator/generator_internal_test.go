package generator

import (
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/stretchr/testify/assert"
)

// TestManifestBindingMatchesCompatSpecs checks that a manifest binding equal to the
// built-in spec for a type produces byte-identical output, so manifests can move to
// bindings without changing generated projects.
func TestManifestBindingMatchesCompatSpecs(t *testing.T) {
	for typ, spec := range appResourceSpecs {
		t.Run(typ, func(t *testing.T) {
			fields := map[string]manifest.ResourceField{}
			values := map[string]string{}
			for _, f := range spec.varFields {
				fields[f[0]] = manifest.ResourceField{Env: "ENV_" + f[0], Description: f[0] + " field"}
				values["res."+f[0]] = "v-" + f[0]
			}
			compat := manifest.Resource{Type: typ, ResourceKey: "res", Description: "desc", Fields: fields}
			bound := compat
			bound.Binding = &manifest.ResourceBinding{YamlKey: spec.yamlKey, VarFields: spec.varFields, StaticFields: spec.staticFields}

			cfg := Config{ResourceValues: values}
			compatPlugins := []manifest.Plugin{{Name: "p", Resources: manifest.Resources{Required: []manifest.Resource{compat}}}}
			boundPlugins := []manifest.Plugin{{Name: "p", Resources: manifest.Resources{Required: []manifest.Resource{bound}}}}

			assert.NotEmpty(t, GenerateBundleResources(compatPlugins, cfg))
			assert.Equal(t, GenerateBundleResources(compatPlugins, cfg), GenerateBundleResources(boundPlugins, cfg))
			assert.Equal(t, GenerateBundleVariables(compatPlugins, cfg), GenerateBundleVariables(boundPlugins, cfg))
			assert.Equal(t, GenerateTargetVariables(compatPlugins, cfg), GenerateTargetVariables(boundPlugins, cfg))
			assert.Equal(t, GenerateAppEnv(compatPlugins, cfg), GenerateAppEnv(boundPlugins, cfg))
		})
	}
}

package generator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// sdkAppResourceKeys returns the resource binding keys of apps.AppResource, which are its
// pointer-to-struct fields (e.g. sql_warehouse, uc_securable), keyed by their json tag.
func sdkAppResourceKeys(t *testing.T) map[string]bool {
	keys := make(map[string]bool)
	typ := reflect.TypeFor[apps.AppResource]()
	for f := range typ.Fields() {
		if f.Type.Kind() != reflect.Pointer || f.Type.Elem().Kind() != reflect.Struct {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		keys[name] = true
	}
	require.NotEmpty(t, keys)
	return keys
}

// TestBindingYamlKeysMatchSDK anchors binding yamlKeys to the Go SDK. A wrong yamlKey in
// both the manifest and appResourceSpecs passes the byte-identical test but fails here.
func TestBindingYamlKeysMatchSDK(t *testing.T) {
	keys := sdkAppResourceKeys(t)

	for typ, spec := range appResourceSpecs {
		assert.True(t, keys[spec.yamlKey], "appResourceSpecs[%q].yamlKey %q is not an apps.AppResource field", typ, spec.yamlKey)
	}

	m, err := manifest.Load("../../../acceptance/apps/init/auth-mode/template")
	require.NoError(t, err)
	var checked int
	for _, p := range m.GetPlugins() {
		for _, r := range append(p.Resources.Required, p.Resources.Optional...) {
			if r.Binding != nil {
				checked++
				assert.True(t, keys[r.Binding.YamlKey], "fixture binding yamlKey %q of %q is not an apps.AppResource field", r.Binding.YamlKey, r.Key())
			}
		}
	}
	require.Positive(t, checked)
}

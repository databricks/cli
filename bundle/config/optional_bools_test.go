package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// plainBools are the bool fields of CLI-defined config structs that predate
// TestConfigBoolsArePointers. Do not add to this list: declare new fields as *bool.
var plainBools = map[string]string{
	"config.Bundle.Force": "readonly: set from --force, not by users",

	// These structs declare ForceSendFields for more than this field, so an
	// explicit false is kept through it.
	"config.Presets.ArtifactsDynamicVersion":     "ForceSendFields also keeps the other presets' zero values",
	"resources.Lifecycle.PreventDestroy":         "ForceSendFields also keeps an explicit depends_on: []",
	"resources.DashboardConfig.EmbedCredentials": "ForceSendFields is passed on to the dashboards API",
}

// Every bool field of a CLI-defined config struct must be a *bool. A plain bool
// loses an explicit false: the typed-to-dynamic conversion that runs after every
// mutator drops zero values, so the false never shows in the config, the bundle
// bitmap cannot record it, and a target cannot override true with false.
func TestConfigBoolsArePointers(t *testing.T) {
	found := map[string]bool{}
	findPlainBools(reflect.TypeFor[Root](), map[reflect.Type]bool{}, found)

	for name := range found {
		_, ok := plainBools[name]
		assert.True(t, ok, "%s is a bool; declare it as *bool so that an explicit false is kept", name)
	}
	for name := range plainBools {
		assert.True(t, found[name], "%s is no longer a bool; remove it from plainBools", name)
	}
}

// findPlainBools records the bool fields of the CLI-defined structs reachable from typ.
func findPlainBools(typ reflect.Type, seen map[reflect.Type]bool, found map[string]bool) {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || seen[typ] {
		return
	}
	seen[typ] = true
	cliDefined := strings.HasPrefix(typ.PkgPath(), "github.com/databricks/cli/")
	for f := range typ.Fields() {
		if !f.IsExported() || f.Tag.Get("json") == "-" {
			continue
		}
		findPlainBools(f.Type, seen, found)
		if cliDefined && f.Type.Kind() == reflect.Bool {
			found[typ.String()+"."+f.Name] = true
		}
	}
}

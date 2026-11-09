package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/assert"
)

// An omitempty bool in a CLI-defined config struct must be a *bool, unless the
// struct declares ForceSendFields. Otherwise an explicit false is lost: the
// typed-to-dynamic conversion that runs after every mutator drops zero values, so
// the false never shows in the config and a target cannot override true with false.
func TestOmitemptyBoolsArePointers(t *testing.T) {
	// replace_existing is a resource input passed to the API as is; it only
	// matters when true.
	skip := map[reflect.Type]bool{
		reflect.TypeFor[resources.PostgresEndpointConfig](): true,
		reflect.TypeFor[resources.PostgresDatabaseConfig](): true,
		reflect.TypeFor[resources.PostgresRoleConfig]():     true,
	}
	checkOmitemptyBools(t, reflect.TypeFor[Root](), skip, map[reflect.Type]bool{})
}

func checkOmitemptyBools(t *testing.T, typ reflect.Type, skip, seen map[reflect.Type]bool) {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || seen[typ] {
		return
	}
	seen[typ] = true
	fsf, ok := typ.FieldByName("ForceSendFields")
	keepsZeros := ok && len(fsf.Index) == 1
	cliDefined := strings.HasPrefix(typ.PkgPath(), "github.com/databricks/cli/") && !skip[typ]
	for f := range typ.Fields() {
		checkOmitemptyBools(t, f.Type, skip, seen)
		// Readonly fields are set by the CLI, not by users.
		if structtag.BundleTag(f.Tag.Get("bundle")).ReadOnly() {
			continue
		}
		if cliDefined && !keepsZeros && f.Type.Kind() == reflect.Bool && strings.Contains(f.Tag.Get("json"), ",omitempty") {
			assert.Fail(t, "omitempty bool", "%s.%s must be a *bool so that an explicit false is kept", typ.Name(), f.Name)
		}
	}
}

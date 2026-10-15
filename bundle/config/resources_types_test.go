package config

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

func TestResourcesTypesMap(t *testing.T) {
	assert.Greater(t, len(ResourcesTypes), 10, "expected ResourcesTypes to have more than 10 entries")

	typ, ok := ResourcesTypes["jobs"]
	assert.True(t, ok, "resources type for 'jobs' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[resources.Job](), typ, "resources type for 'jobs' mismatch")

	typ, ok = ResourcesTypes["jobs.permissions"]
	assert.True(t, ok, "resources type for 'jobs.permissions' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeFor[[]resources.JobPermission](), typ, "resources type for 'jobs.permissions' mismatch")
}

// TestNoSameDepthJSONShadows uses structwalk.WalkType — the dumb walker that
// visits every field including duplicates — to detect unresolved same-depth json
// name collisions. A collision is any json name WalkType visits more than once
// for a resource type, with no depth-0 direct field on the resource struct
// resolving the ambiguity.
//
// structwalk.WalkType is the right tool because it faithfully reproduces the
// traversal structwalk.Walk uses at the value level: it visits every promoted
// field from every anonymous embed, even when two embeds declare the same name.
// Any same-depth collision therefore appears as a path visited twice.
func TestNoSameDepthJSONShadows(t *testing.T) {
	rt := reflect.TypeFor[Resources]()
	var collisions []string

	for f := range rt.Fields() {
		et := f.Type.Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct {
			continue
		}
		group := structtag.JSONTag(f.Tag.Get("json")).Name()

		// depth0 is the set of json names declared as direct (non-anonymous)
		// fields on the resource struct itself. A depth-0 field wins over any
		// same-depth collision at deeper embedding levels, so those are OK.
		depth0 := map[string]bool{}
		for sf := range et.Fields() {
			if sf.Anonymous || sf.PkgPath != "" {
				continue
			}
			name := structtag.JSONTag(sf.Tag.Get("json")).Name()
			if name == "" {
				name = sf.Name
			}
			if name != "-" {
				depth0[name] = true
			}
		}

		// Walk the type and count how many times each top-level json name is
		// visited. structwalk.WalkType is a dumb walker: it visits both
		// declarations when two anonymous embeds carry the same json name,
		// naturally surfacing the collision.
		visits := map[string]int{}
		_ = structwalk.WalkType(et, func(path *structpath.PatternNode, _ reflect.Type, sf *reflect.StructField) bool {
			if sf == nil || path == nil {
				return true
			}
			p := strings.TrimPrefix(path.String(), ".")
			// Only top-level names collide at the resource level.
			if p != "" && !strings.Contains(p, ".") {
				visits[p]++
			}
			return true
		})

		for name, count := range visits {
			if count > 1 && !depth0[name] {
				collisions = append(collisions,
					fmt.Sprintf("%s.%s visited %d times by structwalk with no depth-0 field resolving it",
						group, name, count))
			}
		}
	}

	slices.Sort(collisions)
	assert.Empty(t, collisions,
		"same-depth json name collisions found — encoding/json calls these ambiguous "+
			"and serializes neither; structaccess cannot read or write them either")
}

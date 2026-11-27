package direct

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
)

// A whole-member change on a keyed slice is a two-segment path, so it matches any deeper
// per-field pattern through the short-path branch of matchesFieldRuleBidirectional. A
// per-field recreate rule therefore recreates the parent when a member is merely added or
// removed, which is why a keyed-slice resource needs a carve-out before classification.
func TestKeyedMemberChangeMatchesPerFieldPattern(t *testing.T) {
	// configs/<type>.yml: materialized_features[*].feature_name
	pattern := structpath.NewPatternDotString(nil, "materialized_features")
	pattern = structpath.NewPatternBracketStar(pattern)
	perField := structpath.NewPatternDotString(pattern, "feature_name")

	// structdiff emits this for a member present on one side only.
	member := structpath.NewDotString(nil, "materialized_features")
	member = structpath.NewKeyValue(member, "feature_name", "main.ns.feat")

	assert.Equal(t, 3, perField.Len())
	assert.Equal(t, 2, member.Len())
	assert.False(t, member.HasPatternPrefix(perField), "member is shorter, so not a direct prefix match")
	assert.True(t, matchesFieldRuleBidirectional(member, perField),
		"whole-member change still matches the per-field rule via the truncated pattern")
}

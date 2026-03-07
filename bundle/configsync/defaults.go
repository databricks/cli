package configsync

import "github.com/databricks/cli/libs/structs/structpath"

type resetRule struct {
	field *structpath.PatternNode
	value any
}

// resetValues defines values that should replace CLI-defaulted fields.
// If a CLI-defaulted field is changed on remote and should be disabled
// (e.g. queueing disabled → remote field is nil), we can't define it
// in the config as "null" because the CLI default will be applied again.
var resetValues = map[string][]resetRule{
	"jobs": {
		{field: structpath.MustParsePattern("queue"), value: map[string]any{"enabled": false}},
	},
}

func resetValueIfNeeded(resourceType string, path *structpath.PathNode, value any) any {
	for _, rule := range resetValues[resourceType] {
		if path.HasPatternPrefix(rule.field) {
			return rule.value
		}
	}
	return value
}

package configsync

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
)

// skipByLifecycle reports whether the direct engine's lifecycle metadata says the
// remote value of a change must never be written back to config: a field under
// ignore_remote_changes (output-only, input-only or backend-managed), or a backend
// default the config does not set. The plan action is not enough on its own: it
// is still an update when the config has an undeployed edit to such a field, or
// when a resource's OverrideChangeDesc promotes it (a genie space etag).
func skipByLifecycle(resourceKey, fieldPath string, cd *deployplan.ChangeDesc) bool {
	path, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return false
	}
	resourceType := config.GetResourceTypeFromKey(resourceKey)
	backendDefault := cd.New == nil && cd.Remote != nil && !isJobRunAsKey(resourceKey+"."+fieldPath)
	for _, rc := range []*dresources.ResourceLifecycleConfig{
		dresources.GetResourceConfig(resourceType),
		dresources.GetGeneratedResourceConfig(resourceType),
	} {
		for _, rule := range rc.IgnoreRemoteChanges {
			if path.HasPatternPrefix(rule.Field) {
				return true
			}
		}
		if backendDefault && rc.IsBackendDefault(path, cd.Remote) {
			return true
		}
	}
	return false
}

package configsync

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func revertCliDefaults(b *bundle.Bundle, path string, cd *deployplan.ChangeDesc) *deployplan.ChangeDesc {
	if path == "queue" && cd.Remote == nil {
		isDefaultEnabled := false
		if cd.New != nil && cd.New.(*jobs.QueueSettings).Enabled {
			isDefaultEnabled = true
		}

		if isDefaultEnabled {
			cd = &deployplan.ChangeDesc{
				Old: nil,
				New: nil,
				Remote: &jobs.QueueSettings{
					Enabled: false,
				},
				Action: cd.Action,
				Reason: cd.Reason,
			}
		}
	}

	if path == "max_concurrent_runs" {
		maxConcurrentRunsDefault := b.Config.Presets.JobsMaxConcurrentRuns

		isDefault := false
		if cd.Old != nil && cd.New != nil {
			oldVal, oldOk := cd.Old.(int)
			newVal, newOk := cd.New.(int)
			if oldOk && newOk && oldVal == maxConcurrentRunsDefault && newVal == maxConcurrentRunsDefault {
				isDefault = true
			}
		}

		if isDefault {
			cd = &deployplan.ChangeDesc{
				Old:    nil,
				New:    nil,
				Remote: cd.Remote,
				Action: cd.Action,
				Reason: cd.Reason,
			}
		}
	}

	if path == "tags" {
		tagsDefault := b.Config.Presets.Tags
		if cd.Old != nil && cd.New != nil {
			newVal, newOk := cd.New.(map[string]string)
			if newOk && isMapsStringsEqual(newVal, tagsDefault) {
				cd = &deployplan.ChangeDesc{
					Old:    nil,
					New:    nil,
					Remote: cd.Remote,
					Action: cd.Action,
					Reason: cd.Reason,
				}
			}
		}
	}

	return cd
}

func isMapsStringsEqual(m1 map[string]string, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for key, value := range m1 {
		if m2[key] != value {
			return false
		}
	}
	return true
}

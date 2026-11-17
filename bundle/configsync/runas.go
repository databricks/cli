package configsync

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
)

// The plan skips a job's run_as keys that the config lacks as backend defaults.
// run_as holds a single principal key, so when the remote switches kind (for
// example user_name to service_principal_name) the new key is the edit itself,
// and dropping it while removing the old key leaves "run_as: {}".
const (
	jobRunAsPattern      = "resources.jobs.*.run_as"
	jobRunAsChildPattern = "resources.jobs.*.run_as.*"
	jobsSelectorPrefix   = "jobs:"
)

// RunAsSelection holds the jobs, as "jobs:<id>", whose remote run_as is synced
// even when the config does not set one.
type RunAsSelection []string

// ParseRunAsSelection validates the --sync-run-as-ids selectors.
func ParseRunAsSelection(selectors []string) (RunAsSelection, error) {
	for _, selector := range selectors {
		id, ok := strings.CutPrefix(selector, jobsSelectorPrefix)
		if !ok || id == "" {
			return nil, fmt.Errorf("invalid --sync-run-as-ids value %q, expected jobs:<id> (e.g. jobs:123456789)", selector)
		}
	}
	return selectors, nil
}

// ResourceKeys maps the selected jobs to their plan keys using the index from
// IndexDeployedResources. A job this bundle has not deployed is ignored: it has
// no changes to sync.
func (s RunAsSelection) ResourceKeys(ctx context.Context, deployed map[string]string) map[string]bool {
	keys := make(map[string]bool, len(s))
	for _, selector := range s {
		key, ok := deployed[selector]
		if !ok {
			log.Debugf(ctx, "config-remote-sync: --sync-run-as-ids %s is not deployed by this bundle", selector)
			continue
		}
		keys[key] = true
	}
	return keys
}

// isJobRunAsPath reports whether path is a job's run_as or one of its keys.
// Unlike other backend defaults, these are synced when the config lacks them:
// the keys always, the whole object only as decided by skipJobRunAs.
func isJobRunAsPath(path string) bool {
	return matchPattern(jobRunAsPattern, path) || matchPattern(jobRunAsChildPattern, path)
}

// keepSkippedRunAsChange reports whether a run_as change the plan skipped as a
// backend default should still be considered for syncing.
func keepSkippedRunAsChange(path string, cd *deployplan.ChangeDesc) bool {
	return cd.Reason == deployplan.ReasonBackendDefault && isJobRunAsPath(path)
}

// skipJobRunAs reports whether a change to the run_as object as a whole must
// not be synced. Changes to its keys are synced like any other field.
//
// A remote run_as the config does not set can be a Jobs UI edit or a value the
// server kept: jobs/reset without run_as keeps the previous one. Only the
// caller can tell them apart, so it is synced only for selected jobs.
func skipJobRunAs(path string, cd *deployplan.ChangeDesc, selected bool) bool {
	if !matchPattern(jobRunAsPattern, path) {
		return false
	}
	// The config sets run_as, so the plan reports the remote edit key by key.
	if cd.New != nil {
		return true
	}
	// The Jobs UI always submits a principal, so a removed remote run_as is not
	// an edit to follow.
	if cd.Remote == nil {
		return true
	}
	return !selected
}

package configsync

import (
	"cmp"
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// The plan skips a job's run_as keys that the config lacks as backend defaults.
// run_as holds a single principal key, so when the remote switches kind (for
// example user_name to service_principal_name) the new key is the edit itself,
// and dropping it while removing the old key leaves "run_as: {}".
const (
	runAsField           = "run_as"
	jobRunAsPattern      = "resources.jobs.*." + runAsField
	jobRunAsChildPattern = jobRunAsPattern + ".*"
)

// errPermissionsNotRead reports that a job's ACL was not read, so its owner is unknown.
var errPermissionsNotRead = errors.New("permissions were not read")

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
// A config without run_as means the job runs as its owner, and the Jobs UI
// shows that owner until a principal is picked. So a remote run_as naming the
// owner says what the config already says and is not synced. Any other remote
// run_as is the identity the job runs as, whether it was set in the Jobs UI or
// kept by the server after run_as was removed from the config (jobs/reset
// without run_as keeps the previous one), so it is synced.
func skipJobRunAs(path string, cd *deployplan.ChangeDesc, owner string) bool {
	if !matchPattern(jobRunAsPattern, path) {
		return false
	}
	// The config sets run_as, so the plan reports the remote edit key by key.
	if cd.New != nil {
		return true
	}
	runAs, ok := remoteRunAs(cd.Remote)
	// The Jobs UI always submits a principal, so a removed remote run_as is not
	// an edit to follow.
	if !ok {
		return true
	}
	// Without a known owner, a remote run_as can't be told apart from the default.
	return owner == "" || cmp.Or(runAs.UserName, runAs.ServicePrincipalName, runAs.GroupName) == owner
}

func remoteRunAs(remote any) (jobs.JobRunAs, bool) {
	switch v := remote.(type) {
	case jobs.JobRunAs:
		return v, true
	case *jobs.JobRunAs:
		if v != nil {
			return *v, true
		}
	}
	return jobs.JobRunAs{}, false
}

// jobOwners returns, keyed by plan key, the owner of each job with a remote
// run_as the config does not set: the principal the job runs as without one.
// A job whose ACL could not be read is left out. unmanaged holds the ACLs read
// by readRemotePermissions.
func jobOwners(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, unmanaged map[string][]dresources.StatePermission) map[string]string {
	owners := make(map[string]string)
	for resourceKey, entry := range plan.Plan {
		job, ok := entry.RemoteState.(*dresources.JobRemote)
		cd := entry.Changes[runAsField]
		if !ok || job == nil || cd == nil || cd.New != nil || cd.Remote == nil {
			continue
		}
		acl, err := jobACL(ctx, b, plan, resourceKey, unmanaged)
		if err != nil {
			log.Debugf(ctx, "config-remote-sync: not syncing the run_as of %s, its owner is unknown: %v", resourceKey, err)
			continue
		}
		owners[resourceKey] = jobOwner(acl, job.CreatorUserName)
	}
	return owners
}

// jobACL returns a job's remote ACL. Permissions that the config declares but
// were never deployed have no remote state in the plan, so they are read here.
func jobACL(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, resourceKey string, unmanaged map[string][]dresources.StatePermission) ([]dresources.StatePermission, error) {
	permissionsKey := resourceKey + ".permissions"
	entry := plan.Plan[permissionsKey]
	if entry == nil {
		// Read with the unmanaged permissions; missing if that read failed.
		acl, ok := unmanaged[resourceKey]
		if !ok {
			return nil, errPermissionsNotRead
		}
		return acl, nil
	}
	if state, ok := entry.RemoteState.(*dresources.PermissionsState); ok && state != nil {
		return state.EmbeddedSlice, nil
	}
	adapter := b.DeploymentBundle.Adapters[config.GetResourceTypeFromKey(permissionsKey)]
	if adapter == nil {
		return nil, errPermissionsNotRead
	}
	return readPermissions(ctx, b, adapter, resourceKey)
}

// jobOwner returns the principal a job without run_as runs as: the IS_OWNER
// entry of its ACL or, when job access control is off, its creator. See
// run_as_user_name in the Jobs API.
func jobOwner(acl []dresources.StatePermission, creator string) string {
	for _, p := range acl {
		if p.Level == iam.PermissionLevelIsOwner {
			return cmp.Or(p.UserName, p.ServicePrincipalName)
		}
	}
	return creator
}

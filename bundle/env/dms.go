package env

import "context"

// DMSVariable names the environment variable that turns on deployment history
// recording without setting experimental.record_deployment_history in the bundle.
// It exists for the CLI's own acceptance tests, which run the whole bundle suite
// with DMS enabled: setting it here beats adding the field to every databricks.yml.
//
// Like ForceAllowRecordDeploymentHistoryVariable it is deliberately undocumented; see
// validate.ValidateRecordDeploymentHistory for why the feature is still gated off.
const DMSVariable = "DATABRICKS_BUNDLE_DMS"

// DMS reports whether the environment turns on deployment history recording.
func DMS(ctx context.Context) bool {
	value, ok := get(ctx, []string{DMSVariable})
	return ok && value != "" && value != "0" && value != "false"
}

// RecordsDeploymentHistory reports whether this deploy records deployment history,
// from either the bundle setting or DMSVariable. It is the single predicate the
// recording code paths branch on, so the env var and the config field cannot drift.
func RecordsDeploymentHistory(ctx context.Context, configured bool) bool {
	return configured || DMS(ctx)
}

// DMSAllowExistingResourcesVariable names the environment variable that lets a bundle
// with resources already in its state file be recorded, which is otherwise refused
// (see dstate.DMSSource.AllowExistingResources for what that costs).
const DMSAllowExistingResourcesVariable = "DATABRICKS_BUNDLE_DMS_ALLOW_EXISTING_RESOURCES"

// DMSAllowExistingResources reports whether the environment allows recording a bundle
// that already tracks resources.
func DMSAllowExistingResources(ctx context.Context) bool {
	value, ok := get(ctx, []string{DMSAllowExistingResourcesVariable})
	return ok && value != "" && value != "0" && value != "false"
}

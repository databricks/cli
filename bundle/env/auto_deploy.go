package env

import "context"

// AutoDeployResultFileVariable names the file that `bundle deploy` writes its result to. The
// auto-deploy runner sets it and reads the file after the CLI exits.
const AutoDeployResultFileVariable = "DATABRICKS_AUTO_DEPLOY_RESULT_FILE"

// AutoDeployResultFile returns the path to write the deploy result to.
func AutoDeployResultFile(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		AutoDeployResultFileVariable,
	})
}

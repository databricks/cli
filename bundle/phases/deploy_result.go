package phases

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
)

// deployResult is what a deploy reports in the result file. Unknown fields are left out.
type deployResult struct {
	DeploymentID string `json:"deployment_id,omitempty"`
	VersionName  string `json:"version_name,omitempty"`
}

// WriteDeployResult writes the deployment this deploy read or created, and the version it created,
// to the file named by DATABRICKS_AUTO_DEPLOY_RESULT_FILE. The caller runs it on every exit path,
// failures included, so a missing file means the CLI crashed. b is nil when no bundle was loaded.
func WriteDeployResult(ctx context.Context, b *bundle.Bundle) {
	path, _ := env.AutoDeployResultFile(ctx)
	if path == "" {
		return
	}

	var result deployResult
	if b != nil {
		result.DeploymentID = b.Config.Bundle.Deployment.DeploymentID
		db := &b.DeploymentBundle.StateDB
		// Buffer stats exist only once this run created a version (InitializeOperationBuffer);
		// before that, VersionID is the last version some earlier deploy recorded.
		if db.OperationBufferStats() != nil {
			result.VersionName = dms.VersionName(db.DeploymentID, db.VersionID)
		}
	}

	data, err := json.Marshal(result)
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	if err != nil {
		log.Warnf(ctx, "Failed to write the deploy result to %s: %s", filepath.ToSlash(path), err)
	}
}

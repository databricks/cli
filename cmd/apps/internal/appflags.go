package internal

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/apps/prompt"
)

// Resource name constants for databricks.yml resource bindings.
// These names are used in resource.name fields in databricks.yml and must match
// the keys used in environment variable resolution.
const (
	ResourceNameSQLWarehouse    = "sql-warehouse"
	ResourceNameServingEndpoint = "serving-endpoint"
	ResourceNameExperiment      = "experiment"
	ResourceNameDatabase        = "database"
	ResourceNameDatabaseName    = "database-name"
	ResourceNameUCVolume        = "uc-volume"
)

// ParseDeployAndRunFlags parses the deploy and run flag values into typed values.
func ParseDeployAndRunFlags(deploy bool, run string) (bool, prompt.RunMode, error) {
	var runMode prompt.RunMode
	switch run {
	case "dev":
		runMode = prompt.RunModeDev
	case "dev-remote":
		runMode = prompt.RunModeDevRemote
	case "", "none":
		runMode = prompt.RunModeNone
	default:
		return false, prompt.RunModeNone, fmt.Errorf("invalid --run value: %q (must be none, dev, or dev-remote)", run)
	}

	// dev-remote requires --deploy because it needs a deployed app to connect to
	if runMode == prompt.RunModeDevRemote && !deploy {
		return false, prompt.RunModeNone, errors.New("--run=dev-remote requires --deploy (dev-remote needs a deployed app to connect to)")
	}

	return deploy, runMode, nil
}

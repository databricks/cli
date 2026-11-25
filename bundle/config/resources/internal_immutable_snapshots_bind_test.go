package resources_test

import "github.com/databricks/cli/bundle/config/resources"

func init() {
	registerNotBindable(&resources.Snapshot{},
		"an internal resource that the CLI creates on deploy; it cannot be declared in a bundle, so there is nothing to bind")
}

package resources_test

import "github.com/databricks/cli/bundle/config/resources"

func init() {
	registerNotBindable(&resources.MlflowModel{},
		"binding legacy MLflow models has not been verified; this test has exempted them since it was added")
}

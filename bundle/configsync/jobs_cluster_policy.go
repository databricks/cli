package configsync

import "github.com/databricks/cli/bundle/direct/dresources"

func init() {
	registerClusterPolicyRemote[*dresources.JobRemote]("jobs")
}

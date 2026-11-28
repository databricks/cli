package root

import (
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/databricks/sdk-go/core/clientinfo"
)

func init() {
	useragent.WithProduct("cli", build.GetInfo().Version)
	// The classic SDK panics above on an invalid version, so this cannot fail.
	_ = clientinfo.SetProduct("cli", build.GetInfo().Version)
}

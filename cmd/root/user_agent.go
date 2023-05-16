package root

import (
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go/useragent"
)

func init() {
	useragent.WithProduct("cli", build.GetInfo().Version)
}

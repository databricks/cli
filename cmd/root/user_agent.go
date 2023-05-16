package root

import (
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go/useragent"
)

func init() {
	useragent.WithProduct("bricks", build.GetInfo().Version)
}

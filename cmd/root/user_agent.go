package root

import (
	"github.com/databricks/bricks/internal/build"
	"github.com/databricks/databricks-sdk-go/useragent"
)

func init() {
	useragent.WithProduct("bricks", build.GetInfo().Version)
}

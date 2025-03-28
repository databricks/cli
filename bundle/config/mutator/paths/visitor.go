package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

type VisitFunc func(path dyn.Path, mode TranslateMode, value dyn.Value) (dyn.Value, error)

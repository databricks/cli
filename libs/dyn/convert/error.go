package convert

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

type TypeError struct {
	value dyn.Value
	msg   string
}

func (e TypeError) Error() string {
	return fmt.Sprintf("%s: %s", e.value.Location(), e.msg)
}

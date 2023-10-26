package convert

import (
	"fmt"

	"github.com/databricks/cli/libs/config"
)

type TypeError struct {
	value config.Value
	msg   string
}

func (e TypeError) Error() string {
	return fmt.Sprintf("%s: %s", e.value.Location(), e.msg)
}

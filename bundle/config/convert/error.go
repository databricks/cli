package convert

import "github.com/databricks/cli/libs/config"

type TypeError struct {
	value config.Value
	msg   string
}

package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

type Converter interface {
	Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error
}

var converters = map[string]Converter{}

func GetConverter(name string) (Converter, bool) {
	c, ok := converters[name]
	return c, ok
}

func registerConverter(name string, c Converter) {
	converters[name] = c
}

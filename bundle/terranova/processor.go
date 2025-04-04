package terranova

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type Processor struct {
	Select []string
	Drop   []string
	// TODO: custom func (Value) Value for complex cases
}

func (p *Processor) ApplyProcessor(v dyn.Value) (dyn.Value, error) {
	if p.Select != nil {
		return merge.Select(v, p.Select)
	}
	if p.Drop != nil {
		return merge.AntiSelect(v, p.Drop)
	}
	return v, nil
}

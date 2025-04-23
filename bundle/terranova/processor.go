package terranova

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type Move struct {
	Fields []string
	Target string
}

// Example:
// Input: {"job_id": 123, "field1": "hello", "field2": "world"}, Fields: ["field1", "field2"], Target: "data", Result: {"job_id": 123, "data": {"field1": "hello", "field2": "world"}}
func (p *Move) ApplyMove(v dyn.Value) (dyn.Value, error) {
	// AI TODO: implement this function; add test cases in processor_test.go

}

type Processor struct {
	Select []string
	Drop   []string
	Moves  []Move
	// TODO: custom func (Value) Value for complex cases
}

func (p *Processor) ApplyProcessor(v dyn.Value) (dyn.Value, error) {
	if p.Select != nil {
		v, err := merge.Select(v, p.Select)
		if err != nil {
			return v, err
		}
	}

	if p.Drop != nil {
		v, err := merge.AntiSelect(v, p.Drop)
		if err != nil {
			return v, err
		}
	}

	for _, move := range p.Moves {
		v, err := move.ApplyMove(v)
		if err != nil {
			return v, err
		}
	}

	return v, nil
}

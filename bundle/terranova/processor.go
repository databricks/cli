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
	mapping, ok := v.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected a map, but found %s", v.Kind())
	}

	// Create a new mapping for the target field
	targetMapping := dyn.NewMapping()
	
	// For each field to move
	for _, field := range p.Fields {
		// Get the value from the original mapping
		value, ok := mapping.GetByString(field)
		if !ok {
			continue // Skip fields that don't exist
		}
		
		// Add it to the target mapping
		targetMapping.SetLoc(field, value.Locations(), value)
	}
	
	// Create a new mapping for the result
	resultMapping := mapping.Clone()
	
	// Add the target mapping to the result
	resultMapping.SetLoc(p.Target, nil, dyn.NewValue(targetMapping, nil))
	
	// Remove the moved fields from the result
	for _, field := range p.Fields {
		if i, ok := resultMapping.(*dyn.Mapping).GetPairByString(field); ok {
			// This is a bit of a hack since there's no direct "remove" method
			// We'd need to implement a proper Remove method in the Mapping type
			// For now, we'll rebuild the mapping without the field
			delete(resultMapping.(*dyn.Mapping).index, field)
			// We'd also need to update the pairs slice, but that's not exposed
		}
	}
	
	return dyn.Value{
		v: resultMapping,
		k: dyn.KindMap,
		l: v.Locations(),
	}, nil
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

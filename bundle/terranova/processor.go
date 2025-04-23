package terranova

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

type Move struct {
	Fields []string
	Target string
}

// Example:
// Input: {"job_id": 123, "field1": "hello", "field2": "world"}, Fields: ["field1", "field2"], Target: "data", Result: {"job_id": 123, "data": {"field1": "hello", "field2": "world"}}
// Input: {"job_id": 123, "field1": "hello", "field2": "world"}, Fields: ["!job_id"], Target: "data", Result: {"job_id": 123, "data": {"field1": "hello", "field2": "world"}}
func (p *Move) ApplyMove(v dyn.Value) (dyn.Value, error) {
	mapping, ok := v.AsMap()
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("expected a map, but found %s", v.Kind())
	}

	// Create a new mapping for the target field
	targetMapping := dyn.NewMapping()

	// Create a new result mapping
	resultMapping := dyn.NewMapping()

	// Check if we have any negated fields (starting with !)
	hasNegatedFields := false
	negatedFields := make([]string, 0)
	for _, field := range p.Fields {
		if len(field) > 0 && field[0] == '!' {
			hasNegatedFields = true
			negatedFields = append(negatedFields, field[1:])
		}
	}

	// Process all fields
	for _, pair := range mapping.Pairs() {
		key := pair.Key.MustString()

		shouldMove := false
		if hasNegatedFields {
			// If we have negated fields, move everything EXCEPT those fields
			shouldMove = !slices.Contains(negatedFields, key)
		} else {
			// Otherwise use the normal inclusion logic
			shouldMove = slices.Contains(p.Fields, key)
		}

		if shouldMove {
			targetMapping.SetLoc(key, pair.Key.Locations(), pair.Value)
		} else {
			resultMapping.SetLoc(key, pair.Key.Locations(), pair.Value)
		}
	}

	// Add the target mapping to the result
	resultMapping.SetLoc(p.Target, nil, dyn.NewValue(targetMapping, nil))

	return dyn.NewValue(resultMapping, v.Locations()), nil
}

type Processor struct {
	Select []string
	Drop   []string
	Moves  []Move
	// TODO: custom func (Value) Value for complex cases
}

func (p *Processor) ApplyProcessor(v dyn.Value) (dyn.Value, error) {
	var err error

	if p.Select != nil {
		v, err = merge.Select(v, p.Select)
		if err != nil {
			return v, err
		}
	}

	if p.Drop != nil {
		v, err = merge.AntiSelect(v, p.Drop)
		if err != nil {
			return v, err
		}
	}

	for _, move := range p.Moves {
		v, err = move.ApplyMove(v)
		if err != nil {
			return v, err
		}
	}

	return v, nil
}

func ApplyProcessors(processors []Processor, value dyn.Value) (dyn.Value, error) {
	var err error
	for _, p := range processors {
		value, err = p.ApplyProcessor(value)
		if err != nil {
			return value, err
		}
	}
	return value, nil
}

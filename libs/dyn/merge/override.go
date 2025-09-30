package merge

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// OverrideVisitor is visiting the changes during the override process
// and allows to control what changes are allowed, or update the effective
// value.
//
// For instance, it can disallow changes outside the specific path(s), or update
// the location of the effective value.
//
// Values returned by 'VisitInsert' and 'VisitUpdate' are used as the final value
// of the node. 'VisitDelete' can return ErrOverrideUndoDelete to undo delete.
//
// 'VisitDelete' is called when a value is removed from mapping or sequence
// 'VisitInsert' is called when a new value is added to mapping or sequence
// 'VisitUpdate' is called when a leaf value is updated
type OverrideVisitor struct {
	VisitDelete func(valuePath dyn.Path, left dyn.Value) error
	VisitInsert func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error)
	VisitUpdate func(valuePath dyn.Path, left, right dyn.Value) (dyn.Value, error)
}

var ErrOverrideUndoDelete = errors.New("undo delete operation")

// Override overrides value 'leftRoot' with 'rightRoot', keeping 'location' if values
// haven't changed. Preserving 'location' is important to preserve the original source of the value
// for error reporting.
func Override(leftRoot, rightRoot dyn.Value, visitor OverrideVisitor) (dyn.Value, error) {
	return override(dyn.EmptyPath, leftRoot, rightRoot, visitor)
}

func override(basePath dyn.Path, left, right dyn.Value, visitor OverrideVisitor) (dyn.Value, error) {
	if left.Kind() != right.Kind() {
		return visitor.VisitUpdate(basePath, left, right)
	}

	// NB: we only call 'VisitUpdate' on leaf values, and for sequences and mappings
	// we don't know if value was updated or not

	switch left.Kind() {
	case dyn.KindMap:
		merged, err := overrideMapping(basePath, left.MustMap(), right.MustMap(), visitor)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return dyn.NewValue(merged, left.Locations()), nil

	case dyn.KindSequence:
		// some sequences are keyed, and we can detect which elements are added/removed/updated,
		// but we don't have this information
		merged, err := overrideSequence(basePath, left.MustSequence(), right.MustSequence(), visitor)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return dyn.NewValue(merged, left.Locations()), nil

	case dyn.KindString:
		if left.MustString() == right.MustString() {
			return left, nil
		} else {
			return visitor.VisitUpdate(basePath, left, right)
		}

	case dyn.KindFloat:
		// TODO consider comparison with epsilon if normalization doesn't help, where do we use floats?

		if left.MustFloat() == right.MustFloat() {
			return left, nil
		} else {
			return visitor.VisitUpdate(basePath, left, right)
		}

	case dyn.KindBool:
		if left.MustBool() == right.MustBool() {
			return left, nil
		} else {
			return visitor.VisitUpdate(basePath, left, right)
		}

	case dyn.KindTime:
		if left.MustTime() == right.MustTime() {
			return left, nil
		} else {
			return visitor.VisitUpdate(basePath, left, right)
		}

	case dyn.KindInt:
		if left.MustInt() == right.MustInt() {
			return left, nil
		} else {
			return visitor.VisitUpdate(basePath, left, right)
		}
	case dyn.KindNil:
		return left, nil
	}

	return dyn.InvalidValue, fmt.Errorf("unexpected kind %s at %s", left.Kind(), basePath.String())
}

func overrideMapping(basePath dyn.Path, leftMapping, rightMapping dyn.Mapping, visitor OverrideVisitor) (dyn.Mapping, error) {
	out := dyn.NewMapping()

	for _, leftPair := range leftMapping.Pairs() {
		// detect if key was removed
		if _, ok := rightMapping.GetPair(leftPair.Key); !ok {
			key := leftPair.Key.MustString()
			keyLoc := leftPair.Key.Locations()
			path := basePath.Append(dyn.Key(key))

			err := visitor.VisitDelete(path, leftPair.Value)

			// if 'delete' was undone, add it back
			if errors.Is(err, ErrOverrideUndoDelete) {
				out.SetLoc(key, keyLoc, leftPair.Value)
			} else if err != nil {
				return dyn.NewMapping(), err
			}
		}
	}

	// iterating only right mapping will remove keys not present anymore
	// and insert new keys

	for _, rightPair := range rightMapping.Pairs() {
		key := rightPair.Key.MustString()
		keyLoc := rightPair.Key.Locations()
		if leftPair, ok := leftMapping.GetPair(rightPair.Key); ok {
			path := basePath.Append(dyn.Key(key))
			newValue, err := override(path, leftPair.Value, rightPair.Value, visitor)
			if err != nil {
				return dyn.NewMapping(), err
			}

			// key was there before, so keep its location
			out.SetLoc(key, keyLoc, newValue)
		} else {
			path := basePath.Append(dyn.Key(rightPair.Key.MustString()))

			newValue, err := visitor.VisitInsert(path, rightPair.Value)
			if err != nil {
				return dyn.NewMapping(), err
			}

			out.SetLoc(key, keyLoc, newValue)
		}
	}

	return out, nil
}

func overrideSequence(basePath dyn.Path, left, right []dyn.Value, visitor OverrideVisitor) ([]dyn.Value, error) {
	minLen := min(len(left), len(right))
	var values []dyn.Value

	for i := range minLen {
		path := basePath.Append(dyn.Index(i))
		merged, err := override(path, left[i], right[i], visitor)
		if err != nil {
			return nil, err
		}

		values = append(values, merged)
	}

	if len(right) > len(left) {
		for i := minLen; i < len(right); i++ {
			path := basePath.Append(dyn.Index(i))
			newValue, err := visitor.VisitInsert(path, right[i])
			if err != nil {
				return nil, err
			}

			values = append(values, newValue)
		}
	} else if len(left) > len(right) {
		for i := minLen; i < len(left); i++ {
			path := basePath.Append(dyn.Index(i))
			err := visitor.VisitDelete(path, left[i])

			// if 'delete' was undone, add it back
			if errors.Is(err, ErrOverrideUndoDelete) {
				values = append(values, left[i])
			} else if err != nil {
				return nil, err
			}
		}
	}

	return values, nil
}

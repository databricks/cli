package bitmap

import (
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/dyn"
)

// MarkChanges sets the bits of what after holds that before does not: fields
// before lacks, values that differ, and the fields on the way to them. Both are
// rooted at the configuration root. A list element equal to any element of
// before is unchanged wherever it moved; a changed one is compared with the
// element of before that has the same key field (see listKeys), or, without
// one, with the element at the same position.
func (b *Builder) MarkChanges(before, after dyn.Value) {
	if b == nil {
		return
	}
	b.markChangedChildren(before, after, "")
}

// markChanged marks after at pattern where it differs from before, and reports
// whether it marked anything.
func (b *Builder) markChanged(before, after dyn.Value, pattern string) bool {
	if _, ok := b.index[pattern]; !ok || after.Kind() == dyn.KindNil {
		return false
	}
	switch {
	case before.Kind() != after.Kind():
		b.mark(after, pattern, false)
		return true
	case after.Kind() == dyn.KindMap || after.Kind() == dyn.KindSequence:
		if !b.markChangedChildren(before, after, pattern) {
			return false
		}
		setBit(b.index, b.bits, pattern)
		return true
	case before.AsAny() == after.AsAny():
		return false
	default:
		b.mark(after, pattern, false)
		return true
	}
}

// markChangedChildren is markChanged for the map entries and list elements of
// after, and reports whether it marked any.
func (b *Builder) markChangedChildren(before, after dyn.Value, pattern string) bool {
	changed := false
	if m, ok := after.AsMap(); ok {
		for _, pair := range m.Pairs() {
			key := pair.Key.MustString()
			changed = b.markChanged(before.Get(key), pair.Value, b.childPattern(pattern, key)) || changed
		}
	}
	if s, ok := after.AsSequence(); ok {
		prev, _ := before.AsSequence()
		for i, elem := range s {
			// Unchanged, wherever it moved: a list Python reorders or prepends to
			// keeps the elements it received. The same position is checked first so
			// that an unchanged list costs one comparison per element.
			if i < len(prev) && equalValues(prev[i], elem) ||
				slices.ContainsFunc(prev, func(old dyn.Value) bool { return equalValues(old, elem) }) {
				continue
			}
			changed = b.markChanged(previousElement(prev, elem, i), elem, pattern+listSuffix) || changed
		}
	}
	return changed
}

// listKeys are the fields that identify an element of a keyed list (job tasks,
// job clusters, environments, pipeline clusters, parameters, app resources).
var listKeys = []string{"task_key", "job_cluster_key", "environment_key", "label", "name"}

// previousElement returns the element of prev that elem is a changed version
// of: the one with the same key when elem has a key field, otherwise the one at
// the same position. A keyed element with no match is new.
func previousElement(prev []dyn.Value, elem dyn.Value, i int) dyn.Value {
	for _, key := range listKeys {
		id, ok := elem.Get(key).AsString()
		if !ok {
			continue
		}
		for _, old := range prev {
			if oldID, ok := old.Get(key).AsString(); ok && oldID == id {
				return old
			}
		}
		return dyn.InvalidValue
	}
	if i < len(prev) {
		return prev[i]
	}
	return dyn.InvalidValue
}

// equalValues reports whether a and b hold the same configuration, ignoring
// locations.
func equalValues(a, b dyn.Value) bool {
	return a.Kind() == b.Kind() && reflect.DeepEqual(a.AsAny(), b.AsAny())
}

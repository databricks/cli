package configsync

import (
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/registry"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// A keyed-slice element is addressed in a change path as [='value']: the key field is
// omitted because resolution is by value. configsync works on dynamic values, so it
// recovers the element's key fields from the Go type of the sequence — resolved
// against the bundle config schema — rather than guessing from field names (a task,
// for example, carries both task_key and a job_cluster_key reference, so a name-based
// guess would be ambiguous).

// keyFieldsAtPath returns the key field JSON names of the element type of the sequence
// at seqPath (relative to the bundle config root), or nil if seqPath is not a keyed
// slice.
func keyFieldsAtPath(b *bundle.Bundle, seqPath *structpath.PathNode) []string {
	seqType, err := structaccess.TypeAtPath(reflect.TypeOf(b.Config), seqPath)
	if err != nil {
		return nil
	}
	for seqType.Kind() == reflect.Pointer {
		seqType = seqType.Elem()
	}
	if seqType.Kind() != reflect.Slice && seqType.Kind() != reflect.Array {
		return nil
	}
	return registry.KeyFields(seqType.Elem())
}

// dynElementKey returns the identity of a dynamic keyed-slice element: the value of its
// first non-empty key field, using keyFields resolved from the type. ok is false when
// the element is not a mapping or no key field is set.
func dynElementKey(elem dyn.Value, keyFields []string) (value string, ok bool) {
	for _, field := range keyFields {
		v, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(field)})
		if err != nil || v.Kind() != dyn.KindString {
			continue
		}
		if s := v.MustString(); s != "" {
			return s, true
		}
	}
	return "", false
}

// singleKeyFieldAt returns the sole key field of the element addressed by keyedNode
// (a [='value'] selector, relative to resourceKey), or "" if the element type has no
// key field or more than one. Rename handling uses it to name the key it must strip
// and rewrite; a lone key field (task_key, name, …) is what rename applies to.
func singleKeyFieldAt(b *bundle.Bundle, resourceKey string, keyedNode *structpath.PathNode) string {
	seqPath := keyedNode.Parent()
	if seqPath == nil {
		return ""
	}
	fullSeq, err := structpath.ParsePath(resourceKey + "." + seqPath.String())
	if err != nil {
		return ""
	}
	if kf := keyFieldsAtPath(b, fullSeq); len(kf) == 1 {
		return kf[0]
	}
	return ""
}

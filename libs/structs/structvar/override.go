package structvar

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
)

// ErrOverrideUndoDelete can be returned by [OverrideVisitor.VisitDelete] to keep the
// deleted value.
var ErrOverrideUndoDelete = errors.New("undo delete operation")

// OverrideVisitor is notified of the differences found by [PlanOverride].
// Any error aborts the override.
type OverrideVisitor struct {
	VisitDelete func(path *structpath.PathNode, left View) error
	VisitInsert func(path *structpath.PathNode, right View) error
	VisitUpdate func(path *structpath.PathNode, left, right View) error
}

// OverridePlan is the result of [PlanOverride]; apply it with [StructVar.Override].
type OverridePlan struct {
	src   View
	locs  *Locations
	undos []undo
}

type undo struct {
	path  *structpath.PathNode
	value *StructVar
}

// PlanOverride compares dst with src like merge.Override and reports the differences
// to the visitor. Applying the plan replaces dst with src, except that values unchanged
// in src keep their locations in dst, and values whose deletion the visitor undid are kept.
func PlanOverride(dst, src View, visitor OverrideVisitor) (*OverridePlan, error) {
	plan := &OverridePlan{src: src}
	locs, err := plan.override(nil, dst, src, visitor)
	if err != nil {
		return nil, err
	}
	plan.locs = locs
	return plan, nil
}

func (p *OverridePlan) override(path *structpath.PathNode, left, right View, visitor OverrideVisitor) (*Locations, error) {
	lk, rk := left.Kind(), right.Kind()
	if lk != rk {
		return right.loc, visitor.VisitUpdate(path, left, right)
	}

	switch lk {
	case KindMap:
		out := (*Locations)(nil).WithLocations(left.loc.Get())
		for k, lv := range left.MapItems() {
			if right.Get(k).IsValid() {
				continue
			}
			kp := structpath.NewStringKey(path, k)
			err := visitor.VisitDelete(kp, lv)
			if errors.Is(err, ErrOverrideUndoDelete) {
				p.keep(kp, lv)
				out.setChild(pathComponent{key: k, isKey: true}, lv.loc)
			} else if err != nil {
				return nil, err
			}
		}
		for k, rv := range right.MapItems() {
			kp := structpath.NewStringKey(path, k)
			locs := rv.loc
			if lv := left.Get(k); lv.IsValid() {
				var err error
				locs, err = p.override(kp, lv, rv, visitor)
				if err != nil {
					return nil, err
				}
			} else if err := visitor.VisitInsert(kp, rv); err != nil {
				return nil, err
			}
			out.setChild(pathComponent{key: k, isKey: true}, locs)
		}
		return out, nil
	case KindSequence:
		out := (*Locations)(nil).WithLocations(left.loc.Get())
		var ls, rs []View
		for _, v := range left.Sequence() {
			ls = append(ls, v)
		}
		for _, v := range right.Sequence() {
			rs = append(rs, v)
		}
		n := 0
		for i := range min(len(ls), len(rs)) {
			locs, err := p.override(structpath.NewIndex(path, i), ls[i], rs[i], visitor)
			if err != nil {
				return nil, err
			}
			out.setChild(pathComponent{index: n}, locs)
			n++
		}
		for i := len(ls); i < len(rs); i++ {
			if err := visitor.VisitInsert(structpath.NewIndex(path, i), rs[i]); err != nil {
				return nil, err
			}
			out.setChild(pathComponent{index: n}, rs[i].loc)
			n++
		}
		for i := len(rs); i < len(ls); i++ {
			err := visitor.VisitDelete(structpath.NewIndex(path, i), ls[i])
			if errors.Is(err, ErrOverrideUndoDelete) {
				p.keep(structpath.NewIndex(path, n), ls[i])
				out.setChild(pathComponent{index: n}, ls[i].loc)
				n++
			} else if err != nil {
				return nil, err
			}
		}
		return out, nil
	case KindNil:
		return left.loc, nil
	case KindString, KindBool, KindInt, KindFloat:
		if left.AsAny() == right.AsAny() || sameNumber(left, right) {
			return left.loc, nil
		}
		return right.loc, visitor.VisitUpdate(path, left, right)
	default:
		return nil, fmt.Errorf("unexpected kind %s at %s", lk, path)
	}
}

func sameNumber(a, b View) bool {
	if ai, ok := a.AsInt(); ok {
		bi, ok := b.AsInt()
		return ok && ai == bi
	}
	return false
}

// keep records that the value at path (in the result) is the kept left value. It is
// copied now, because applying the plan overwrites the left side.
func (p *OverridePlan) keep(path *structpath.PathNode, v View) {
	cp := &StructVar{Value: reflect.New(v.v.Type()).Interface()}
	_, _ = cp.Assign(nil, v)
	p.undos = append(p.undos, undo{path: path, value: cp})
}

// Override replaces sv.Value with the result of plan (see [PlanOverride]).
func (sv *StructVar) Override(plan *OverridePlan) error {
	sv.Refs = nil
	sv.Locations = nil
	if _, err := sv.Assign(nil, plan.src); err != nil {
		return err
	}
	for _, u := range plan.undos {
		if _, err := sv.Assign(u.path, u.value.View()); err != nil {
			return err
		}
	}
	sv.Locations = plan.locs
	return nil
}

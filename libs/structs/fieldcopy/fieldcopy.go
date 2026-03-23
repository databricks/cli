// Package fieldcopy provides reflection-based struct-to-struct field copying
// with test-time completeness validation. Field mappings are computed once
// on first use and cached for subsequent calls.
//
// If both Src and Dst have a ForceSendFields []string field, it is handled
// automatically: source values are filtered to only include names of exported
// Dst fields that are being copied (not in SkipDst).
package fieldcopy

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

const forceSendFieldsName = "ForceSendFields"

// Copy describes a field-by-field mapping from Src to Dst struct type.
// Fields with matching names and assignable types are copied automatically.
type Copy[Src, Dst any] struct {
	// Rename maps destination field name to source field name for fields
	// with different names in source and destination.
	Rename map[string]string

	// SkipSrc lists source field names intentionally not copied.
	SkipSrc []string

	// SkipDst lists destination field names intentionally left at zero value.
	SkipDst []string

	once   sync.Once
	copyFn func(src *Src) Dst
}

// Do copies fields from src to a new Dst value using precomputed field mappings.
// On first call, field mappings are computed via reflection and cached.
func (c *Copy[Src, Dst]) Do(src *Src) Dst {
	c.once.Do(func() {
		c.copyFn = c.build()
	})
	return c.copyFn(src)
}

type fieldOp struct {
	dstIndex []int
	srcIndex []int
}

func (c *Copy[Src, Dst]) build() func(*Src) Dst {
	srcType := reflect.TypeFor[Src]()
	dstType := reflect.TypeFor[Dst]()

	skipDst := toSet(c.SkipDst)

	// Detect auto-handling of ForceSendFields: both types must have it as []string
	// and it must not be in SkipDst (which would mean manual handling).
	var autoFSF bool
	var fsfSrcIdx, fsfDstIdx []int
	var validFSFNames map[string]bool

	stringSliceType := reflect.TypeFor[[]string]()
	if !skipDst[forceSendFieldsName] {
		dstFSF, dstOK := dstType.FieldByName(forceSendFieldsName)
		srcFSF, srcOK := srcType.FieldByName(forceSendFieldsName)
		if dstOK && srcOK && dstFSF.Type == stringSliceType && srcFSF.Type == stringSliceType {
			autoFSF = true
			fsfSrcIdx = srcFSF.Index
			fsfDstIdx = dstFSF.Index

			// Build set of valid field names: exported dst fields that are being copied.
			validFSFNames = make(map[string]bool)
			for i := range dstType.NumField() {
				f := dstType.Field(i)
				if f.IsExported() && f.Name != forceSendFieldsName && !skipDst[f.Name] {
					validFSFNames[f.Name] = true
				}
			}
		}
	}

	var ops []fieldOp

	for i := range dstType.NumField() {
		df := dstType.Field(i)
		if !df.IsExported() || skipDst[df.Name] {
			continue
		}
		if autoFSF && df.Name == forceSendFieldsName {
			continue // handled separately
		}

		srcName := df.Name
		if c.Rename != nil {
			if renamed, ok := c.Rename[df.Name]; ok {
				srcName = renamed
			}
		}

		sf, ok := srcType.FieldByName(srcName)
		if !ok || !sf.Type.AssignableTo(df.Type) {
			continue
		}

		ops = append(ops, fieldOp{dstIndex: df.Index, srcIndex: sf.Index})
	}

	return func(src *Src) Dst {
		var dst Dst
		sv := reflect.ValueOf(src).Elem()
		dv := reflect.ValueOf(&dst).Elem()
		for _, op := range ops {
			dv.FieldByIndex(op.dstIndex).Set(sv.FieldByIndex(op.srcIndex))
		}
		if autoFSF {
			srcFSF := sv.FieldByIndex(fsfSrcIdx)
			if !srcFSF.IsNil() {
				srcFields := srcFSF.Interface().([]string)
				var filtered []string
				for _, name := range srcFields {
					if validFSFNames[name] {
						filtered = append(filtered, name)
					}
				}
				if filtered != nil {
					dv.FieldByIndex(fsfDstIdx).Set(reflect.ValueOf(filtered))
				}
			}
		}
		return dst
	}
}

// Validate checks that the mapping is complete:
//   - Every exported Dst field is either matched in Src, in Rename, or in SkipDst.
//   - Every exported Src field is either matched in Dst, a Rename source, or in SkipSrc.
//   - No stale entries in SkipSrc, SkipDst, or Rename.
//
// ForceSendFields is considered auto-handled when both types have it as []string
// and it is not in SkipDst.
func (c *Copy[Src, Dst]) Validate() error {
	srcType := reflect.TypeFor[Src]()
	dstType := reflect.TypeFor[Dst]()

	skipSrc := toSet(c.SkipSrc)
	skipDst := toSet(c.SkipDst)

	// Detect auto-handled ForceSendFields.
	autoFSF := false
	stringSliceType := reflect.TypeFor[[]string]()
	if !skipDst[forceSendFieldsName] {
		dstFSF, dstOK := dstType.FieldByName(forceSendFieldsName)
		srcFSF, srcOK := srcType.FieldByName(forceSendFieldsName)
		if dstOK && srcOK && dstFSF.Type == stringSliceType && srcFSF.Type == stringSliceType {
			autoFSF = true
		}
	}

	// Build set of rename targets (src field names used by Rename).
	renameTargets := make(map[string]string) // src name → dst name
	if c.Rename != nil {
		for dstName, srcName := range c.Rename {
			renameTargets[srcName] = dstName
		}
	}

	var errs []string

	// Check every exported dst field is handled.
	for i := range dstType.NumField() {
		df := dstType.Field(i)
		if !df.IsExported() {
			continue
		}
		if skipDst[df.Name] {
			continue
		}
		if autoFSF && df.Name == forceSendFieldsName {
			continue
		}

		srcName := df.Name
		if c.Rename != nil {
			if renamed, ok := c.Rename[df.Name]; ok {
				srcName = renamed
			}
		}

		sf, ok := srcType.FieldByName(srcName)
		if !ok {
			errs = append(errs, fmt.Sprintf("dst field %q: no matching field %q on %v (add to Rename or SkipDst)", df.Name, srcName, srcType))
			continue
		}
		if !sf.Type.AssignableTo(df.Type) {
			errs = append(errs, fmt.Sprintf("dst field %q: type %v not assignable from src type %v (add to SkipDst and handle in post-processing)", df.Name, df.Type, sf.Type))
		}
	}

	// Check every exported src field is consumed.
	for i := range srcType.NumField() {
		sf := srcType.Field(i)
		if !sf.IsExported() {
			continue
		}
		if skipSrc[sf.Name] {
			continue
		}
		if autoFSF && sf.Name == forceSendFieldsName {
			continue
		}
		if _, ok := renameTargets[sf.Name]; ok {
			continue
		}
		if _, ok := dstType.FieldByName(sf.Name); ok {
			continue
		}
		errs = append(errs, fmt.Sprintf("src field %q on %v not consumed (add to dst %v, Rename, or SkipSrc)", sf.Name, srcType, dstType))
	}

	// Check for stale SkipSrc entries.
	for _, name := range c.SkipSrc {
		if _, ok := srcType.FieldByName(name); !ok {
			errs = append(errs, fmt.Sprintf("stale SkipSrc entry %q: field does not exist on %v", name, srcType))
		}
	}

	// Check for stale SkipDst entries.
	for _, name := range c.SkipDst {
		if _, ok := dstType.FieldByName(name); !ok {
			errs = append(errs, fmt.Sprintf("stale SkipDst entry %q: field does not exist on %v", name, dstType))
		}
	}

	// Check for stale Rename entries.
	if c.Rename != nil {
		for dstName, srcName := range c.Rename {
			if _, ok := dstType.FieldByName(dstName); !ok {
				errs = append(errs, fmt.Sprintf("stale Rename key %q: field does not exist on %v", dstName, dstType))
			}
			if _, ok := srcType.FieldByName(srcName); !ok {
				errs = append(errs, fmt.Sprintf("stale Rename value %q: field does not exist on %v", srcName, srcType))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("fieldcopy.Validate[%v → %v]:\n  %s", srcType, dstType, strings.Join(errs, "\n  "))
	}
	return nil
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

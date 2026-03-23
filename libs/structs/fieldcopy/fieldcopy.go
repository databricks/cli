// Package fieldcopy provides reflection-based struct-to-struct field copying
// with test-time completeness validation via golden files.
//
// Field mappings are computed once on first use and cached for subsequent calls.
// If both Src and Dst have a ForceSendFields []string field, it is handled
// automatically: source values are filtered to only include names of exported
// Dst fields that are being copied.
package fieldcopy

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const forceSendFieldsName = "ForceSendFields"

// Copy describes a field-by-field mapping from Src to Dst struct type.
// Fields with matching names and assignable types are copied automatically.
// Unmatched fields are left at zero and reported via [Copy.Report] for
// golden file testing.
type Copy[Src, Dst any] struct {
	// Rename maps destination field name to source field name for fields
	// with different names in source and destination.
	Rename map[string]string

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

	// Detect auto-handling of ForceSendFields: both types must have it as []string.
	var autoFSF bool
	var fsfSrcIdx, fsfDstIdx []int
	var validFSFNames map[string]bool

	stringSliceType := reflect.TypeFor[[]string]()
	dstFSF, dstOK := dstType.FieldByName(forceSendFieldsName)
	srcFSF, srcOK := srcType.FieldByName(forceSendFieldsName)
	if dstOK && srcOK && dstFSF.Type == stringSliceType && srcFSF.Type == stringSliceType {
		autoFSF = true
		fsfSrcIdx = srcFSF.Index
		fsfDstIdx = dstFSF.Index
		validFSFNames = make(map[string]bool)
	}

	var ops []fieldOp

	for i := range dstType.NumField() {
		df := dstType.Field(i)
		if !df.IsExported() {
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
		if autoFSF {
			validFSFNames[df.Name] = true
		}
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

// Report returns a human-readable summary of unmatched fields for golden file testing.
// Fields that exist on Src but have no match on Dst are listed as "src not copied".
// Fields that exist on Dst but have no match on Src are listed as "dst not set".
func (c *Copy[Src, Dst]) Report() string {
	srcType := reflect.TypeFor[Src]()
	dstType := reflect.TypeFor[Dst]()
	return c.report(srcType, dstType)
}

func (c *Copy[Src, Dst]) report(srcType, dstType reflect.Type) string {
	// Build rename lookups.
	renameTargets := make(map[string]bool) // src names used by Rename
	if c.Rename != nil {
		for _, srcName := range c.Rename {
			renameTargets[srcName] = true
		}
	}

	// Detect auto-handled ForceSendFields.
	autoFSF := false
	stringSliceType := reflect.TypeFor[[]string]()
	if dstFSF, ok := dstType.FieldByName(forceSendFieldsName); ok {
		if srcFSF, ok := srcType.FieldByName(forceSendFieldsName); ok {
			if dstFSF.Type == stringSliceType && srcFSF.Type == stringSliceType {
				autoFSF = true
			}
		}
	}

	// Collect matched dst field names (including renames).
	matchedDst := make(map[string]bool)
	matchedSrc := make(map[string]bool)
	for i := range dstType.NumField() {
		df := dstType.Field(i)
		if !df.IsExported() {
			continue
		}
		if autoFSF && df.Name == forceSendFieldsName {
			matchedDst[df.Name] = true
			matchedSrc[df.Name] = true
			continue
		}

		srcName := df.Name
		if c.Rename != nil {
			if renamed, ok := c.Rename[df.Name]; ok {
				srcName = renamed
			}
		}

		sf, ok := srcType.FieldByName(srcName)
		if ok && sf.Type.AssignableTo(df.Type) {
			matchedDst[df.Name] = true
			matchedSrc[srcName] = true
		}
	}

	// Find unmatched src fields.
	var unmatchedSrc []string
	for i := range srcType.NumField() {
		sf := srcType.Field(i)
		if !sf.IsExported() || matchedSrc[sf.Name] || renameTargets[sf.Name] {
			continue
		}
		unmatchedSrc = append(unmatchedSrc, sf.Name)
	}

	// Find unmatched dst fields.
	var unmatchedDst []string
	for i := range dstType.NumField() {
		df := dstType.Field(i)
		if !df.IsExported() || matchedDst[df.Name] {
			continue
		}
		unmatchedDst = append(unmatchedDst, df.Name)
	}

	// Check for stale Rename entries.
	var staleRenames []string
	if c.Rename != nil {
		for dstName, srcName := range c.Rename {
			var issues []string
			if _, ok := dstType.FieldByName(dstName); !ok {
				issues = append(issues, fmt.Sprintf("dst %q not found", dstName))
			}
			if _, ok := srcType.FieldByName(srcName); !ok {
				issues = append(issues, fmt.Sprintf("src %q not found", srcName))
			}
			if len(issues) > 0 {
				staleRenames = append(staleRenames, fmt.Sprintf("%s→%s (%s)", dstName, srcName, strings.Join(issues, ", ")))
			}
		}
		sort.Strings(staleRenames)
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "%v → %v", srcType, dstType)

	if len(unmatchedSrc) == 0 && len(unmatchedDst) == 0 && len(staleRenames) == 0 {
		buf.WriteString(": all fields matched\n")
		return buf.String()
	}

	buf.WriteString("\n")

	if len(unmatchedSrc) > 0 {
		buf.WriteString("  src not copied:\n")
		for _, name := range unmatchedSrc {
			fmt.Fprintf(&buf, "    - %s\n", name)
		}
	}
	if len(unmatchedDst) > 0 {
		buf.WriteString("  dst not set:\n")
		for _, name := range unmatchedDst {
			fmt.Fprintf(&buf, "    - %s\n", name)
		}
	}
	if len(staleRenames) > 0 {
		buf.WriteString("  stale renames:\n")
		for _, s := range staleRenames {
			fmt.Fprintf(&buf, "    - %s\n", s)
		}
	}

	return buf.String()
}

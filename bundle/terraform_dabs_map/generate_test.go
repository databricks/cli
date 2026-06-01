package terraform_dabs_map_test

import (
	"fmt"
	"go/format"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/require"
)

// TestGenerateSchemaMap verifies that generated.go is up to date.
// Run with -update to regenerate it.
func TestGenerateSchemaMap(t *testing.T) {
	results, err := buildAll()
	require.NoError(t, err)

	src, err := renderSource(results)
	require.NoError(t, err)

	if testdiff.OverwriteMode {
		require.NoError(t, os.WriteFile("generated.go", src, 0o644))
		return
	}

	existing, err := os.ReadFile("generated.go")
	require.NoError(t, err)
	// Normalize CRLF so the comparison is stable on Windows (Git autocrlf).
	existingNorm := strings.ReplaceAll(string(existing), "\r\n", "\n")
	testdiff.AssertEqualTexts(t, "generated.go", "want", existingNorm, string(src))
}

// tfTypes maps TF resource type name → Go struct type, built from AllResources.
var tfTypes = func() map[string]reflect.Type {
	rt := reflect.TypeFor[schema.AllResources]()
	m := make(map[string]reflect.Type, rt.NumField())
	for f := range rt.Fields() {
		tag := f.Tag.Get("json")
		tfType := strings.SplitN(tag, ",", 2)[0]
		if tfType != "" && tfType != "-" {
			m[tfType] = f.Type
		}
	}
	return m
}()

// dabsKnownFields are top-level DABs fields with no TF equivalent; suppress from output.
var dabsKnownFields = map[string]bool{
	"id":              true,
	"permissions":     true,
	"url":             true,
	"lifecycle":       true,
	"grants":          true,
	"modified_status": true,
}

// tfKnownFields are top-level TF fields with no DABs equivalent; suppress from output.
var tfKnownFields = map[string]bool{
	"id": true,
}

// tfKnownSegments are TF path segments that suppress any path containing them at any level.
var tfKnownSegments = map[string]bool{
	"provider_config": true, // Terraform provider metadata, not a DABs concept
}

type groupResult struct {
	group      string
	tfType     string
	hasTFType  bool
	renames    map[string]string // TF path → DABs path (renamed fields only)
	unwraps    []string          // TF paths that are structural wrappers (Unwrap: true)
	dabsOnly   map[string]bool   // DABs clean paths with no Terraform equivalent
	tfOnly     map[string]bool   // TF clean paths with no DABs equivalent
	matchCount int               // used for stats output only, not written to generated.go
}

func buildAll() ([]groupResult, error) {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil, fmt.Errorf("initialize adapters: %w", err)
	}

	var groups []string
	for group := range terraform.GroupToTerraformName {
		if !strings.Contains(group, ".") {
			groups = append(groups, group)
		}
	}
	slices.Sort(groups)

	var results []groupResult
	for _, group := range groups {
		adapter, ok := adapters[group]
		if !ok {
			continue
		}
		result, err := buildGroup(group, adapter)
		if err != nil {
			return nil, fmt.Errorf("build group %s: %w", group, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func buildGroup(group string, adapter *dresources.Adapter) (groupResult, error) {
	tfType := terraform.GroupToTerraformName[group]
	tfTyp, hasTFType := tfTypes[tfType]

	// Collect DABs clean field paths.
	dabsFields := make(map[string]bool)
	err := structwalk.WalkType(adapter.InputConfigType(), func(path *structpath.PatternNode, _ reflect.Type, _ *reflect.StructField) bool {
		if path == nil {
			return true
		}
		p := strings.TrimPrefix(path.String(), ".")
		// Skip permissions and grants sub-resource fields.
		if p == "permissions" || strings.HasPrefix(p, "permissions.") || strings.HasPrefix(p, "permissions[") ||
			p == "grants" || strings.HasPrefix(p, "grants.") || strings.HasPrefix(p, "grants[") {
			return false
		}
		dabsFields[cleanPath(p)] = true
		return true
	})
	if err != nil {
		return groupResult{}, fmt.Errorf("walk DABs type: %w", err)
	}

	// Collect TF clean field paths.
	tfFields := make(map[string]bool)
	if hasTFType {
		err = structwalk.WalkType(tfTyp, func(path *structpath.PatternNode, _ reflect.Type, _ *reflect.StructField) bool {
			if path == nil {
				return true
			}
			p := strings.TrimPrefix(path.String(), ".")
			tfFields[cleanPath(p)] = true
			return true
		})
		if err != nil {
			return groupResult{}, fmt.Errorf("walk TF type: %w", err)
		}
	}

	res := groupResult{group: group, tfType: tfType, hasTFType: hasTFType}
	if !hasTFType {
		return res, nil
	}

	res.renames = make(map[string]string)
	res.dabsOnly = make(map[string]bool)
	res.tfOnly = make(map[string]bool)

	// Step 1: exact matches.
	matchedDABs := make(map[string]bool)
	matchedTF := make(map[string]bool)
	for dabs := range dabsFields {
		if dabsKnownFields[topSegment(dabs)] {
			continue
		}
		if tfFields[dabs] {
			matchedDABs[dabs] = true
			matchedTF[dabs] = true
			res.matchCount++
		}
	}

	// Step 2: stemmed matches.
	// candidates[tfPath] = DABs paths whose first valid stem maps to tfPath.
	candidates := make(map[string][]string)
	for dabs := range dabsFields {
		if matchedDABs[dabs] || dabsKnownFields[topSegment(dabs)] {
			continue
		}
		if tfPath, ok := matchToTF(dabs, tfFields); ok && !matchedTF[tfPath] {
			candidates[tfPath] = append(candidates[tfPath], dabs)
		}
	}

	for tfPath, dabsPaths := range candidates {
		if len(dabsPaths) > 1 {
			// Multiple DABs fields stem to the same TF path — conflict: keep as dabs_only.
			fmt.Fprintf(os.Stderr, "warning: %s: conflict for TF path %q: %v\n", group, tfPath, dabsPaths)
			continue
		}
		dabsPath := dabsPaths[0]
		matchedDABs[dabsPath] = true
		matchedTF[tfPath] = true
		// Only record in renames when the leaf field name itself differs;
		// child paths of a renamed parent (where only ancestors differ) count as matches.
		if lastSegment(tfPath) != lastSegment(dabsPath) {
			res.renames[tfPath] = dabsPath
		} else {
			res.matchCount++
		}
	}

	// Step 3: detect wrapper segments — TF segments that wrap DABs fields one level deeper
	// with no name change (e.g. DABs "budget_policy_id" ↔ TF "spec.budget_policy_id").
	// A wrapper is accepted only when ALL its sub-fields are accounted for by unmatched DABs
	// fields; this rejects output-only wrappers like "status" that carry extra computed fields.
	wrappers := make(map[string]bool)
	for tf := range tfFields {
		if !matchedTF[tf] {
			if head, _, ok := strings.Cut(tf, "."); ok {
				wrappers[head] = true
			}
		}
	}
	for wrapper := range wrappers {
		prefix := wrapper + "."
		subTF := make(map[string]bool)
		for tf := range tfFields {
			if !matchedTF[tf] {
				if after, ok := strings.CutPrefix(tf, prefix); ok {
					subTF[after] = true
				}
			}
		}
		// Collect unmatched DABs fields that have an exact counterpart in subTF.
		var matching []string
		for dabs := range dabsFields {
			if !matchedDABs[dabs] && !dabsKnownFields[topSegment(dabs)] && subTF[dabs] {
				matching = append(matching, dabs)
			}
		}
		// Only treat as a wrapper when every sub-field is accounted for.
		if len(matching) == 0 || len(matching) != len(subTF) {
			continue
		}
		for _, dabs := range matching {
			matchedDABs[dabs] = true
			matchedTF[prefix+dabs] = true
			res.matchCount++
		}
		matchedTF[wrapper] = true // mark the wrapper segment itself to suppress it from tfOnly
		res.unwraps = append(res.unwraps, wrapper)
	}

	// Step 4: remaining unmatched fields.
	for dabs := range dabsFields {
		if !matchedDABs[dabs] && !dabsKnownFields[topSegment(dabs)] {
			res.dabsOnly[dabs] = true
		}
	}
	for tf := range tfFields {
		if !matchedTF[tf] && !tfKnownFields[topSegment(tf)] && !hasKnownSegment(tf, tfKnownSegments) {
			res.tfOnly[tf] = true
		}
	}

	// Step 5: classify TF-only fields that are accessible via RemoteType as computed
	// (server-generated outputs the direct engine can read, but the user doesn't configure).
	remoteFields := make(map[string]bool)
	err = structwalk.WalkType(adapter.RemoteType(), func(path *structpath.PatternNode, _ reflect.Type, _ *reflect.StructField) bool {
		if path == nil {
			return true
		}
		p := strings.TrimPrefix(path.String(), ".")
		remoteFields[cleanPath(p)] = true
		return true
	})
	if err != nil {
		return groupResult{}, fmt.Errorf("walk remote type: %w", err)
	}
	for tf := range res.tfOnly {
		if remoteFields[tf] {
			delete(res.tfOnly, tf)
		}
	}

	return res, nil
}

// topSegment returns the first dot-separated segment of path.
func topSegment(path string) string {
	if before, _, ok := strings.Cut(path, "."); ok {
		return before
	}
	return path
}

// hasKnownSegment reports whether any segment of path is in known.
func hasKnownSegment(path string, known map[string]bool) bool {
	for seg := path; seg != ""; {
		var head string
		if before, after, ok := strings.Cut(seg, "."); ok {
			head, seg = before, after
		} else {
			head, seg = seg, ""
		}
		if known[head] {
			return true
		}
	}
	return false
}

// lastSegment returns the last dot-separated segment of path.
func lastSegment(path string) string {
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// cleanPath strips [*] array markers and collapses repeated dots.
func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "[*]", "")
	for strings.Contains(p, "..") {
		p = strings.ReplaceAll(p, "..", ".")
	}
	return strings.Trim(p, ".")
}

// stem returns the canonical stemmed form of a single field name: strips a leading
// "git_" prefix, then applies plural→singular (ies→y, or trailing s→"").
// Returns seg unchanged if no transformation applies.
func stem(seg string) string {
	seg = strings.TrimPrefix(seg, "git_")
	if s, ok := strings.CutSuffix(seg, "ies"); ok {
		return s + "y"
	}
	if s, ok := strings.CutSuffix(seg, "s"); ok && s != "" {
		return s
	}
	return seg
}

// matchToTF maps a DABs field path to its TF equivalent by processing one segment
// at a time: each segment is tried as-is and then stemmed, and once a prefix is found
// in tfFields the tail is recursively resolved under that prefix.
// Returns ("", false) when no match exists.
func matchToTF(dabs string, tfFields map[string]bool) (string, bool) {
	head, tail, hasTail := strings.Cut(dabs, ".")
	hvs := []string{head}
	if s := stem(head); s != head {
		hvs = append(hvs, s)
	}
	for _, hv := range hvs {
		if !hasTail {
			if tfFields[hv] {
				return hv, true
			}
			continue
		}
		prefix := hv + "."
		subTF := make(map[string]bool)
		for tf := range tfFields {
			if after, ok := strings.CutPrefix(tf, prefix); ok {
				subTF[after] = true
			}
		}
		if len(subTF) == 0 {
			continue
		}
		if sub, ok := matchToTF(tail, subTF); ok {
			return hv + "." + sub, true
		}
	}
	return "", false
}

func renderSource(results []groupResult) ([]byte, error) {
	var b strings.Builder
	w := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	w("// Code generated by bundle/terraform_dabs_map/generate_test.go; DO NOT EDIT.\n\n")
	w("package terraform_dabs_map\n\n")

	for _, r := range results {
		if !r.hasTFType {
			continue
		}
		for _, c := range []struct {
			label string
			n     int
		}{
			{"renames", len(r.renames)},
			{"dabs-only", len(r.dabsOnly)},
			{"tf-only", len(r.tfOnly)},
			{"unwraps", len(r.unwraps)},
		} {
			if c.n > 0 {
				w("// %s / %s: %d %s\n", r.group, r.tfType, c.n, c.label)
			}
		}
	}
	w("\n")

	w("// TerraformToDABsFieldMap maps DABs group name → nested TF segments → DABs segment name.\n")
	w("// Navigate using TF field name segments; DABs is the corresponding DABs name when it differs.\n")
	w("var TerraformToDABsFieldMap = map[string]RenameTree{\n")
	for _, r := range results {
		if !r.hasTFType || (len(r.renames) == 0 && len(r.unwraps) == 0) {
			continue
		}
		w("\t%q: {\n", r.group)
		writeRenameTree(w, buildRenameTree(r.renames, r.unwraps), 2)
		w("\t},\n")
	}
	w("}\n\n")

	w("// DABsOnlyFields maps DABs group name → FieldSet of DABs fields with no TF equivalent.\n")
	w("var DABsOnlyFields = map[string]FieldSet{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.dabsOnly) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		writeFieldSet(w, buildFieldSet(r.dabsOnly), 2)
		w("\t},\n")
	}
	w("}\n\n")

	w("// TerraformOnlyFields maps DABs group name → FieldSet of TF fields with no DABs equivalent.\n")
	w("var TerraformOnlyFields = map[string]FieldSet{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.tfOnly) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		writeFieldSet(w, buildFieldSet(r.tfOnly), 2)
		w("\t},\n")
	}
	w("}\n\n")

	w("// DABsToTerraformRenameMap maps DABs group name → nested DABs segments → TF segment name.\n")
	w("// Navigate using DABs field name segments; NewName is the TF name when it differs.\n")
	w("var DABsToTerraformRenameMap = map[string]RenameTree{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.renames) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		writeRenameTree(w, buildRenameTree(invertRenames(r.renames), nil), 2)
		w("\t},\n")
	}
	w("}\n\n")

	w("// DABsToTerraformWrappers maps DABs group name → the TF wrapper segment to prepend.\n")
	w("// For groups using Unwrap in TerraformToDABsFieldMap, every DABs path must be prefixed\n")
	w("// with this segment to obtain the corresponding TF path.\n")
	w("var DABsToTerraformWrappers = map[string]string{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.unwraps) == 0 {
			continue
		}
		for _, wrapper := range r.unwraps {
			w("\t%q: %q,\n", r.group, wrapper)
		}
	}
	w("}\n")

	return format.Source([]byte(b.String()))
}

// rnode is an internal node used when building the RenameTree before rendering.
type rnode struct {
	dabs     string
	unwrap   bool
	children map[string]*rnode
}

// invertRenames swaps key and value in a flat rename map, producing the inverse mapping.
func invertRenames(renames map[string]string) map[string]string {
	inv := make(map[string]string, len(renames))
	for k, v := range renames {
		inv[v] = k
	}
	return inv
}

// buildRenameTree converts flat rename mappings and unwrap wrappers to a nested rnode tree.
// At each level it stores the renamed segment name when it differs from the key segment.
func buildRenameTree(renames map[string]string, unwraps []string) map[string]*rnode {
	root := make(map[string]*rnode)
	for tfPath, dabsPath := range renames {
		tfSegs := strings.Split(tfPath, ".")
		dabsSegs := strings.Split(dabsPath, ".")
		cur := root
		for i, seg := range tfSegs {
			if cur[seg] == nil {
				cur[seg] = &rnode{}
			}
			n := cur[seg]
			if i == len(tfSegs)-1 {
				// Leaf level: record the DABs name if it differs.
				if dabsSegs[i] != seg {
					n.dabs = dabsSegs[i]
				}
			} else {
				if n.children == nil {
					n.children = make(map[string]*rnode)
				}
				cur = n.children
			}
		}
	}
	for _, wrapper := range unwraps {
		if root[wrapper] == nil {
			root[wrapper] = &rnode{}
		}
		root[wrapper].unwrap = true
	}
	return root
}

// writeRenameTree writes a rnode tree as RenameTree Go source at the given indent depth.
func writeRenameTree(w func(string, ...any), tree map[string]*rnode, depth int) {
	indent := strings.Repeat("\t", depth)
	for _, key := range slices.Sorted(maps.Keys(tree)) {
		n := tree[key]
		switch {
		case n.unwrap && len(n.children) == 0:
			w("%s%q: {Unwrap: true},\n", indent, key)
		case n.unwrap && len(n.children) > 0:
			w("%s%q: {Unwrap: true, Children: RenameTree{\n", indent, key)
			writeRenameTree(w, n.children, depth+1)
			w("%s}},\n", indent)
		case n.dabs != "" && len(n.children) == 0:
			w("%s%q: {NewName: %q},\n", indent, key, n.dabs)
		case n.dabs == "" && len(n.children) > 0:
			w("%s%q: {Children: RenameTree{\n", indent, key)
			writeRenameTree(w, n.children, depth+1)
			w("%s}},\n", indent)
		default: // both dabs and children
			w("%s%q: {NewName: %q, Children: RenameTree{\n", indent, key, n.dabs)
			writeRenameTree(w, n.children, depth+1)
			w("%s}},\n", indent)
		}
	}
}

// buildFieldSet converts a flat set of dotted paths to a nested map[string]any tree.
// Leaves are represented as empty map[string]any values.
func buildFieldSet(paths map[string]bool) map[string]any {
	root := make(map[string]any)
	for path := range paths {
		cur := root
		for seg := range strings.SplitSeq(path, ".") {
			if cur[seg] == nil {
				cur[seg] = make(map[string]any)
			}
			cur = cur[seg].(map[string]any)
		}
	}
	return root
}

// writeFieldSet writes a field set tree as nested FieldSet Go source at the given depth.
func writeFieldSet(w func(string, ...any), tree map[string]any, depth int) {
	indent := strings.Repeat("\t", depth)
	for _, key := range slices.Sorted(maps.Keys(tree)) {
		child := tree[key].(map[string]any)
		if len(child) == 0 {
			w("%s%q: {},\n", indent, key)
		} else {
			w("%s%q: {\n", indent, key)
			writeFieldSet(w, child, depth+1)
			w("%s},\n", indent)
		}
	}
}

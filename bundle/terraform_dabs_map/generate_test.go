package terraform_dabs_map_test

import (
	"fmt"
	"go/format"
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
		for _, r := range results {
			if r.hasTFType {
				t.Logf("%s (%s): %d matches, %d renames, %d dabs-only, %d tf-only",
					r.group, r.tfType, r.matchCount, len(r.renames), len(r.dabsOnly), len(r.tfOnly))
			}
		}
		return
	}

	existing, err := os.ReadFile("generated.go")
	require.NoError(t, err)
	testdiff.AssertEqualTexts(t, "generated.go", "want", string(existing), string(src))
}

// tfTypes maps TF resource type name → Go struct type, built from ResourceSchemas.
var tfTypes = func() map[string]reflect.Type {
	rt := reflect.TypeFor[schema.ResourceSchemas]()
	m := make(map[string]reflect.Type, rt.NumField())
	for i := range rt.NumField() {
		f := rt.Field(i)
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
	"id":              true,
	"provider_config": true, // Terraform provider metadata, not a DABs concept
}

type groupResult struct {
	group      string
	tfType     string
	hasTFType  bool
	renames    map[string]string // TF path → DABs path (renamed fields only)
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
		for _, stem := range allStems(dabs) {
			if tfFields[stem] && !matchedTF[stem] {
				candidates[stem] = append(candidates[stem], dabs)
				break
			}
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

	// Step 3: remaining unmatched fields.
	for dabs := range dabsFields {
		if !matchedDABs[dabs] && !dabsKnownFields[topSegment(dabs)] {
			res.dabsOnly[dabs] = true
		}
	}
	for tf := range tfFields {
		if !matchedTF[tf] && !tfKnownFields[topSegment(tf)] {
			res.tfOnly[tf] = true
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

// segmentVariants returns all stem variants of a single path segment.
// The original is listed first; variants with fewer transformations come before
// variants with more (suffix-only, prefix-only, then both).
func segmentVariants(seg string) []string {
	vars := []string{seg}

	// Suffix transformation: ies→y or s→"".
	var suffixed string
	if strings.HasSuffix(seg, "ies") {
		suffixed = seg[:len(seg)-3] + "y"
	} else if len(seg) > 1 && strings.HasSuffix(seg, "s") {
		suffixed = seg[:len(seg)-1]
	}
	if suffixed != "" {
		vars = append(vars, suffixed)
	}

	// Prefix transformation: git_→"".
	if strings.HasPrefix(seg, "git_") {
		stripped := seg[4:]
		if stripped != "" {
			vars = append(vars, stripped)
		}
		// Also apply suffix transformation to the stripped form.
		if strings.HasSuffix(stripped, "ies") {
			vars = append(vars, stripped[:len(stripped)-3]+"y")
		} else if len(stripped) > 1 && strings.HasSuffix(stripped, "s") {
			vars = append(vars, stripped[:len(stripped)-1])
		}
	}

	return vars
}

// allStems returns all stemmed path variants, excluding the original.
// Variants with fewer per-segment transformations come first.
func allStems(path string) []string {
	segs := strings.Split(path, ".")
	combos := []string{""}
	for _, seg := range segs {
		variants := segmentVariants(seg)
		var next []string
		for _, prefix := range combos {
			for _, v := range variants {
				if prefix == "" {
					next = append(next, v)
				} else {
					next = append(next, prefix+"."+v)
				}
			}
		}
		combos = next
	}
	// Exclude the original (first combo is always the original).
	var result []string
	for _, c := range combos {
		if c != path {
			result = append(result, c)
		}
	}
	return result
}

func renderSource(results []groupResult) ([]byte, error) {
	var b strings.Builder
	w := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	w("// Code generated by bundle/terraform_dabs_map/generate_test.go; DO NOT EDIT.\n\n")
	w("package terraform_dabs_map\n\n")

	w("// TerraformToDABsFieldMap maps DABs group name → (TF field path → DABs field path)\n")
	w("// for fields that exist in both but under different leaf names. Exact matches are omitted.\n")
	w("var TerraformToDABsFieldMap = map[string]map[string]string{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.renames) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		for _, tf := range sortedKeys(r.renames) {
			w("\t\t%q: %q,\n", tf, r.renames[tf])
		}
		w("\t},\n")
	}
	w("}\n\n")

	w("// DABsOnlyFields maps DABs group name → set of DABs field paths with no Terraform equivalent.\n")
	w("var DABsOnlyFields = map[string]map[string]bool{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.dabsOnly) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		for _, p := range sortedKeys(r.dabsOnly) {
			w("\t\t%q: true,\n", p)
		}
		w("\t},\n")
	}
	w("}\n\n")

	w("// TerraformOnlyFields maps DABs group name → set of TF field paths with no DABs equivalent.\n")
	w("var TerraformOnlyFields = map[string]map[string]bool{\n")
	for _, r := range results {
		if !r.hasTFType || len(r.tfOnly) == 0 {
			continue
		}
		w("\t%q: {\n", r.group)
		for _, p := range sortedKeys(r.tfOnly) {
			w("\t\t%q: true,\n", p)
		}
		w("\t},\n")
	}
	w("}\n")

	return format.Source([]byte(b.String()))
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

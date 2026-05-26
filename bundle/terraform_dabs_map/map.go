// Package terraform_dabs_map builds a bidirectional map between DABs resource
// fields and Terraform resource fields.
package terraform_dabs_map

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// tfTypes maps Terraform resource type names to their generated Go struct types.
// Built from schema.ResourceSchemas via reflection so it stays in sync with the
// generated schema package without a separate manual registry.
var tfTypes = func() map[string]reflect.Type {
	rt := reflect.TypeOf(schema.ResourceSchemas{})
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

// Status of a DABs↔TF field comparison entry.
type Status string

const (
	StatusMatch    Status = "match"
	StatusRenamed  Status = "renamed"
	StatusDabsOnly Status = "dabs_only"
	StatusTFOnly   Status = "tf_only"
	StatusNoTFType Status = "no_tf_type"
)

// Entry is a single DABs↔TF field comparison result.
type Entry struct {
	Status   Status
	DabsPath string // full path e.g. "resources.jobs.*.tasks[*].task_key"
	TFType   string // Terraform resource type e.g. "databricks_job", or "?" if unknown
	TFPath   string // TF field path e.g. "task[*].task_key", or "?" if dabs_only/no_tf_type
}

// dabsKnownFields are top-level DABs resource fields with no TF equivalent.
// Suppress them from dabs_only output to reduce noise.
var dabsKnownFields = map[string]bool{
	"permissions": true,
	"url":         true,
	"lifecycle":   true,
	"grants":      true,
}


// Build returns the complete DABs↔TF schema field comparison.
// It walks DABs resource types from dresources adapters and TF resource types
// from the generated schema structs; no external files are required.
func Build() ([]Entry, error) {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize adapters: %w", err)
	}

	var entries []Entry

	for group, tfType := range terraform.GroupToTerraformName {
		// Skip 3-level groups like "jobs.permissions".
		if strings.Contains(group, ".") {
			continue
		}

		adapter, ok := adapters[group]
		if !ok {
			continue
		}

		// Collect DABs field paths by walking the adapter's input config type.
		// dabsFields maps cleanPath -> fullPath (with [*] notation).
		dabsFields := make(map[string]string)
		err := structwalk.WalkType(adapter.InputConfigType(), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
			if path == nil {
				return true
			}
			p := strings.TrimPrefix(path.String(), ".")
			// Skip permissions and grants sub-resource fields (handled by separate adapters).
			if p == "permissions" || strings.HasPrefix(p, "permissions.") || strings.HasPrefix(p, "permissions[") ||
				p == "grants" || strings.HasPrefix(p, "grants.") || strings.HasPrefix(p, "grants[") {
				return false
			}
			fullPath := p
			cleanPath := cleanFieldPath(p)
			dabsFields[cleanPath] = fullPath
			return true
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk input type for %s: %w", group, err)
		}

		// Collect TF field paths.
		tfTyp, hasTFType := tfTypes[tfType]
		tfFields := make(map[string]bool)
		if hasTFType {
			err = structwalk.WalkType(tfTyp, func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
				if path == nil {
					return true
				}
				p := strings.TrimPrefix(path.String(), ".")
				tfFields[cleanFieldPath(p)] = true
				return true
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk TF type for %s: %w", tfType, err)
			}
		}

		matchedTF := make(map[string]bool)

		for cleanPath, fullPath := range dabsFields {
			dabsPath := "resources." + group + ".*." + fullPath

			if !hasTFType {
				entries = append(entries, Entry{
					Status:   StatusNoTFType,
					DabsPath: dabsPath,
					TFType:   "?",
					TFPath:   "?",
				})
				continue
			}

			tfPath, renamedIdx := applyRenames(group, cleanPath)

			if tfFields[tfPath] {
				matchedTF[tfPath] = true
				if len(renamedIdx) == 0 {
					entries = append(entries, Entry{
						Status:   StatusMatch,
						DabsPath: dabsPath,
						TFType:   tfType,
						TFPath:   tfPath,
					})
				} else {
					// Only emit a renamed entry when the field's own segment is renamed
					// (i.e., the last segment index is in renamedIdx), not just ancestors.
					parts := strings.Split(cleanPath, ".")
					maxRenamed := slices.Max(renamedIdx)
					if maxRenamed == len(parts)-1 {
						entries = append(entries, Entry{
							Status:   StatusRenamed,
							DabsPath: dabsPath,
							TFType:   tfType,
							TFPath:   tfPath,
						})
					}
				}
			} else if tfFields[cleanPath] {
				matchedTF[cleanPath] = true
				entries = append(entries, Entry{
					Status:   StatusMatch,
					DabsPath: dabsPath,
					TFType:   tfType,
					TFPath:   cleanPath,
				})
			} else {
				topField := strings.SplitN(cleanPath, ".", 2)[0]
				if !dabsKnownFields[topField] {
					entries = append(entries, Entry{
						Status:   StatusDabsOnly,
						DabsPath: dabsPath,
						TFType:   tfType,
						TFPath:   "?",
					})
				}
			}
		}

		// TF-only fields.
		for tfPath := range tfFields {
			if !matchedTF[tfPath] {
				entries = append(entries, Entry{
					Status:   StatusTFOnly,
					DabsPath: "?",
					TFType:   tfType,
					TFPath:   tfPath,
				})
			}
		}
	}

	slices.SortFunc(entries, func(a, b Entry) int {
		if a.TFType != b.TFType {
			return strings.Compare(a.TFType, b.TFType)
		}
		if a.DabsPath != b.DabsPath {
			return strings.Compare(a.DabsPath, b.DabsPath)
		}
		return strings.Compare(a.TFPath, b.TFPath)
	})

	return entries, nil
}

// cleanFieldPath strips [*] array markers from a field path and normalises dots.
func cleanFieldPath(p string) string {
	p = strings.ReplaceAll(p, "[*]", "")
	for strings.Contains(p, "..") {
		p = strings.ReplaceAll(p, "..", ".")
	}
	p = strings.Trim(p, ".")
	return p
}

package configsync

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/structs/structpath"
)

type FieldChange struct {
	FilePath        string
	Change          *ConfigChangeDesc
	FieldCandidates []string

	sourceValue         dyn.Value
	sourceSiblings      []dyn.Value
	originalFileContent []byte
}

// resolveSelectors converts key-value selectors to numeric indices that match
// the YAML file positions. It also returns the location of the resolved leaf value.
// Example: "resources.jobs.foo.tasks[task_key='main'].name" -> "resources.jobs.foo.tasks[1].name"
// Returns a PatternNode because for Add operations, [*] may be used as a placeholder for new elements.
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (*structpath.PatternNode, dyn.Location, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return nil, dyn.Location{}, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var result *structpath.PatternNode
	currentValue := b.Config.Value()

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			result = structpath.NewPatternStringKey(result, key)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			result = structpath.NewPatternIndex(result, idx)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return nil, dyn.Location{}, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
			}

			seq, _ := currentValue.AsSequence()
			foundIndex := -1

			for i, elem := range seq {
				keyValue, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(key)})
				if err != nil {
					continue
				}

				if keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					foundIndex = i
					break
				}
			}

			if foundIndex == -1 {
				if operation == OperationAdd {
					result = structpath.NewPatternBracketStar(result)
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return nil, dyn.Location{}, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			// Mutators may reorder sequence elements (e.g., tasks sorted by task_key).
			// Use location information to determine the original YAML file position.
			yamlIndex := yamlFileIndex(seq, foundIndex)
			result = structpath.NewPatternIndex(result, yamlIndex)
			currentValue = seq[foundIndex]
			continue
		}
	}

	return result, currentValue.Location(), nil
}

// yamlFileIndex determines the original YAML file position of a sequence element.
// Mutators may reorder sequence elements (e.g., tasks sorted by task_key), so the
// in-memory index may not match the position in the YAML file. This function uses
// location information to count how many elements from the same file appear before
// the target element, giving the correct index for YAML patching.
func yamlFileIndex(seq []dyn.Value, sortedIndex int) int {
	matchLocation := seq[sortedIndex].Location()
	if matchLocation.File == "" {
		return sortedIndex
	}

	yamlIndex := 0
	for i, elem := range seq {
		if i == sortedIndex {
			continue
		}
		loc := elem.Location()
		if loc.File == matchLocation.File && (loc.Line < matchLocation.Line || (loc.Line == matchLocation.Line && loc.Column < matchLocation.Column)) {
			yamlIndex++
		}
	}
	return yamlIndex
}

func pathDepth(pathStr string) int {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return 0
	}
	return len(node.AsSlice())
}

// ResolveChanges resolves selectors and computes field path candidates for each change.
func ResolveChanges(ctx context.Context, b *bundle.Bundle, configChanges Changes) ([]FieldChange, error) {
	snapshot, err := CaptureSourceSnapshot(ctx, b)
	if err != nil {
		return nil, err
	}
	return ResolveChangesFromSnapshot(ctx, b, configChanges, snapshot)
}

// ResolveChangesFromSnapshot resolves changes against source content captured
// before the remote plan was calculated.
func ResolveChangesFromSnapshot(ctx context.Context, b *bundle.Bundle, configChanges Changes, snapshot *SourceSnapshot) ([]FieldChange, error) {
	if snapshot == nil || snapshot.index == nil {
		return nil, errors.New("source snapshot unavailable")
	}
	return resolveChangesWithProvenance(ctx, b, configChanges, snapshot.index)
}

// translateWorkspacePaths recursively converts absolute workspace paths to relative
// paths when they fall within the bundle's sync root. Paths are made relative to
// targetDir (the directory of the YAML file being patched). For notebook paths
// where the extension was stripped by translate_paths, it restores the extension
// by checking the local filesystem.
func translateWorkspacePaths(value any, syncRootPath string, syncRoot fs.FS, targetDir string) any {
	switch v := value.(type) {
	case string:
		after, ok := strings.CutPrefix(v, syncRootPath+"/")
		if !ok {
			return v
		}
		after = resolveNotebookExtension(syncRoot, after)
		fullPath := filepath.Join(syncRootPath, after)
		relPath, err := filepath.Rel(targetDir, fullPath)
		if err != nil {
			return "./" + after
		}
		relPathSlash := filepath.ToSlash(relPath)
		if !strings.HasPrefix(relPathSlash, "..") {
			relPathSlash = "./" + relPathSlash
		}
		return relPathSlash
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = translateWorkspacePaths(val, syncRootPath, syncRoot, targetDir)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = translateWorkspacePaths(val, syncRootPath, syncRoot, targetDir)
		}
		return result
	default:
		return value
	}
}

// resolveNotebookExtension checks if a relative path refers to a notebook whose
// extension was stripped by translate_paths. If the file doesn't exist as-is but
// exists with a notebook extension, the extension is appended.
func resolveNotebookExtension(syncRoot fs.FS, relPath string) string {
	if syncRoot == nil {
		return relPath
	}
	if _, err := fs.Stat(syncRoot, relPath); err == nil {
		return relPath
	}
	for _, ext := range notebook.Extensions {
		if _, err := fs.Stat(syncRoot, relPath+ext); err == nil {
			return relPath + ext
		}
	}
	return relPath
}

package configsync

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
	"go.yaml.in/yaml/v3"
)

// ApplyChangesToYAML generates YAML files for the given field changes.
func ApplyChangesToYAML(ctx context.Context, b *bundle.Bundle, fieldChanges []FieldChange) ([]FileChange, error) {
	originalFiles := make(map[string][]byte)
	modifiedFiles := make(map[string][]byte)
	fileFieldChanges := make(map[string][]FieldChange)

	for _, fieldChange := range fieldChanges {
		filePath := fieldChange.FilePath

		if _, exists := modifiedFiles[filePath]; !exists {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			originalFiles[filePath] = content
			modifiedFiles[filePath] = preserveBlankLines(content)
		}

		modifiedContent, err := applyChange(ctx, modifiedFiles[filePath], fieldChange)
		if err != nil {
			return nil, fmt.Errorf("failed to apply change to file %s for a field %s: %w", filePath, fieldChange.WritePath, err)
		}

		modifiedFiles[filePath] = modifiedContent
		fileFieldChanges[filePath] = append(fileFieldChanges[filePath], fieldChange)
	}

	var result []FileChange
	for filePath := range modifiedFiles {
		// TODO: A good alternative approach is to remove parent nodes during the Resolve phase,
		// when all of their keys/items are removed, but this should be tested for edge cases.
		// In this case flow style will never appear because empty nodes are never serialized and we won't need clearAddedFlowStyle
		normalized, err := clearAddedFlowStyle(modifiedFiles[filePath], fileFieldChanges[filePath])
		if err != nil {
			return nil, fmt.Errorf("failed to normalize YAML style in %s: %w", filePath, err)
		}
		result = append(result, FileChange{
			Path:            filePath,
			OriginalContent: string(originalFiles[filePath]),
			ModifiedContent: string(restoreBlankLines(normalized)),
		})
	}

	slices.SortFunc(result, func(a, b FileChange) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return result, nil
}

type parentNode struct {
	path        yamlpatch.Path
	missingPath yamlpatch.Path
}

// applyChange applies a single field change to YAML content.
func applyChange(ctx context.Context, content []byte, fieldChange FieldChange) ([]byte, error) {
	success := false
	var firstErr error
	var parentNodesToCreate []parentNode

	for _, fieldPathCandidate := range fieldChange.writePaths() {
		jsonPointer, err := strPathToJSONPointer(fieldPathCandidate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field path %q to JSON pointer: %w", fieldPathCandidate, err)
		}

		path, err := yamlpatch.ParsePath(jsonPointer)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON Pointer %s: %w", jsonPointer, err)
		}

		patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
		var modifiedContent []byte
		var patchErr error

		switch fieldChange.Change.Operation {
		case OperationRemove:
			modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
				Type: yamlpatch.OperationRemove,
				Path: path,
			}})
		case OperationReplace:
			modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
				Type:  yamlpatch.OperationReplace,
				Path:  path,
				Value: fieldChange.Change.Value,
			}})
		case OperationAdd:
			modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
				Type:  yamlpatch.OperationAdd,
				Path:  path,
				Value: fieldChange.Change.Value,
			}})

			if patchErr != nil {
				patched, err := addThroughNullAncestor(content, fieldPathCandidate, fieldChange.Change.Value)
				if err != nil {
					return nil, err
				}
				if patched != nil {
					modifiedContent, patchErr = patched, nil
				}
			}

			// Collect parent path errors for later retry
			if patchErr != nil && isParentPathError(patchErr) {
				if missingPath, extractErr := extractMissingPath(patchErr); extractErr == nil {
					parentNodesToCreate = append(parentNodesToCreate, parentNode{path, missingPath})
				}
			}
		default:
			return nil, fmt.Errorf("unknown operation type %q", fieldChange.Change.Operation)
		}

		if patchErr == nil {
			content = modifiedContent
			log.Debugf(ctx, "Applied changes to %s", jsonPointer)
			success = true
			firstErr = nil
			break
		}

		log.Debugf(ctx, "Failed to apply change to %s: %v", jsonPointer, patchErr)
		if firstErr == nil {
			firstErr = patchErr
		}
	}

	// If all attempts failed with parent path errors, try creating nested structures
	if !success && len(parentNodesToCreate) > 0 {
		for _, errInfo := range parentNodesToCreate {
			nestedValue := buildNestedMaps(errInfo.path, errInfo.missingPath, fieldChange.Change.Value)

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			modifiedContent, patchErr := patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
				Type:  yamlpatch.OperationAdd,
				Path:  errInfo.missingPath,
				Value: nestedValue,
			}})

			if patchErr == nil {
				content = modifiedContent
				firstErr = nil
				log.Debugf(ctx, "Created nested structure at %s", errInfo.missingPath.String())
				break
			}
			if firstErr == nil {
				firstErr = patchErr
			}
			log.Debugf(ctx, "Failed to create nested structure at %s: %v", errInfo.missingPath.String(), patchErr)
		}
	}

	if firstErr != nil {
		if (fieldChange.Change.Operation == OperationRemove || fieldChange.Change.Operation == OperationReplace) && isPathNotFoundError(firstErr) {
			return nil, fmt.Errorf("field not found in YAML configuration: %w", firstErr)
		}
		return nil, fmt.Errorf("failed to apply change: %w", firstErr)
	}

	return content, nil
}

// addThroughNullAncestor applies an add by materializing its null ancestor.
// It returns nil when no null ancestor can be replaced safely.
func addThroughNullAncestor(content []byte, fieldPath string, value any) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) != 1 {
		return nil, nil
	}
	pattern, err := structpath.ParsePattern(fieldPath)
	if err != nil {
		return nil, err
	}
	current := doc.Content[0]
	var keyNode *yaml.Node
	for depth, part := range pattern.AsSlice() {
		// Replacing anchored content could alter aliases or leave dangling references.
		if current.Anchor != "" {
			return nil, nil
		}
		if current.Kind == yaml.ScalarNode && current.ShortTag() == "!!null" {
			for node := pattern; node.Len() > depth; node = node.Parent() {
				key, isKey := node.StringKey()
				index, isIndex := node.Index()
				switch {
				case isKey:
					value = map[string]any{key: value}
				case node.BracketStar(), isIndex && index == 0:
					value = []any{value}
				default:
					// Only the first element can be added to an empty sequence.
					return nil, nil
				}
			}
			var replacement yaml.Node
			if err := replacement.Encode(value); err != nil {
				return nil, err
			}
			replacement.HeadComment = current.HeadComment
			replacement.FootComment = current.FootComment
			// Block containers cannot carry the scalar's inline comment themselves.
			if current.LineComment != "" {
				if keyNode != nil {
					keyNode.LineComment = strings.TrimSpace(keyNode.LineComment + "\n" + current.LineComment)
				} else {
					replacement.HeadComment = strings.TrimSpace(replacement.HeadComment + "\n" + current.LineComment)
				}
			}
			*current = replacement
			return encodeYAML(&doc)
		}

		current, keyNode = yamlChild(current, (*structpath.PathNode)(part))
		if current == nil {
			return nil, nil
		}
	}
	return nil, nil
}

// isParentPathError checks if error indicates missing parent path.
func isParentPathError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "parent path") && strings.Contains(msg, "does not exist")
}

// isPathNotFoundError checks if error indicates the path itself does not exist.
func isPathNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist")
}

// extractMissingPath extracts the missing path from error message like:
// "op add /a/b/c/d: parent path /a/b does not exist"
// Returns: "/a/b"
func extractMissingPath(err error) (yamlpatch.Path, error) {
	msg := err.Error()
	start := strings.Index(msg, "parent path ")
	if start == -1 {
		return nil, errors.New("could not find 'parent path' in error message")
	}
	start += len("parent path ")

	end := strings.Index(msg[start:], " does not exist")
	if end == -1 {
		return nil, errors.New("could not find 'does not exist' in error message")
	}

	pathStr := msg[start : start+end]
	return yamlpatch.ParsePath(pathStr)
}

// buildNestedMaps creates a nested map structure from targetPath to missingPath.
// Example:
//
//	targetPath: /a/b/c/d/e
//	missingPath: /a/b
//	leafValue: "foo"
//
// Returns: {c: {d: {e: "foo"}}}
func buildNestedMaps(targetPath, missingPath yamlpatch.Path, leafValue any) any {
	missingLen := len(missingPath)
	targetLen := len(targetPath)

	if missingLen >= targetLen {
		// Missing path is not a parent of target path
		return leafValue
	}

	// Build nested structure from leaf to missing parent
	result := leafValue
	for i := targetLen - 1; i >= missingLen; i-- {
		result = map[string]any{
			targetPath[i]: result,
		}
	}

	return result
}

// strPathToJSONPointer converts a structpath string to JSON Pointer format.
// Example: "resources.jobs.test[0].name" -> "/resources/jobs/test/0/name"
// The path may contain [*] which is converted to "-" (JSON Pointer append syntax).
func strPathToJSONPointer(pathStr string) (string, error) {
	node, err := structpath.ParsePattern(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse path %q: %w", pathStr, err)
	}

	var parts []string
	for _, n := range node.AsSlice() {
		if key, ok := n.StringKey(); ok {
			parts = append(parts, key)
			continue
		}

		if idx, ok := n.Index(); ok {
			parts = append(parts, strconv.Itoa(idx))
			continue
		}

		// Handle append marker: /-/ is a syntax for appending to an array in JSON Pointer
		if n.BracketStar() {
			parts = append(parts, "-")
			continue
		}

		return "", fmt.Errorf("unsupported path node type in path %q", pathStr)
	}

	if len(parts) == 0 {
		return "", nil
	}
	return "/" + strings.Join(parts, "/"), nil
}

// clearAddedFlowStyle clears FlowStyle on YAML nodes along the changed field paths.
// This prevents flow-style formatting (e.g. {key: value}) that yaml.v3 introduces
// when empty mappings are serialized as "{}" during patch operations
func clearAddedFlowStyle(content []byte, fieldChanges []FieldChange) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return content, nil //nolint:nilerr // return original content if YAML parsing fails
	}
	for _, fc := range fieldChanges {
		for _, candidate := range fc.writePaths() {
			clearFlowStyleAlongPath(&doc, candidate)
		}
	}
	return encodeYAML(&doc)
}

// encodeYAML uses the same indentation as the YAML patcher.
func encodeYAML(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}

// yamlChild returns the child and its mapping key so callers can preserve comments.
// A nil child means the path component cannot be followed.
func yamlChild(current *yaml.Node, part *structpath.PathNode) (child, keyNode *yaml.Node) {
	switch current.Kind {
	case yaml.MappingNode:
		key, ok := part.StringKey()
		if !ok {
			return nil, nil
		}
		// current.Content: [key1, val1, key2, val2, ...]
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == key {
				return current.Content[i+1], current.Content[i]
			}
		}
	case yaml.SequenceNode:
		index, ok := part.Index()
		if ok && index >= 0 && index < len(current.Content) {
			return current.Content[index], nil
		}
	default:
		return nil, nil
	}
	return nil, nil
}

// clearFlowStyleAlongPath navigates the YAML tree along the given structpath,
// clearing FlowStyle on every node from root to leaf (inclusive).
func clearFlowStyleAlongPath(doc *yaml.Node, pathStr string) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return
	}

	current := doc
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	for _, n := range node.AsSlice() {
		current.Style &^= yaml.FlowStyle
		current, _ = yamlChild(current, n)
		if current == nil {
			return
		}
	}

	clearFlowStyleNodes(current)
}

func clearFlowStyleNodes(node *yaml.Node) {
	node.Style &^= yaml.FlowStyle
	for _, child := range node.Content {
		clearFlowStyleNodes(child)
	}
}

const blankLineMarker = "# __YAMLPATCH_BLANK_LINE__"

// blockScalarRe matches YAML lines that start a block scalar (| or >).
// [-:]  — colon (mapping value) or dash (sequence item)
// [|>]  — literal or folded block scalar indicator
// [-+0-9]* — optional chomp (+/-) and indent indicators
// (?:#.*)? — optional comment
var blockScalarRe = regexp.MustCompile(`[-:]\s+[|>][-+0-9]*\s*(?:#.*)?$`)

// preserveBlankLines replaces blank lines with marker comments that survive yaml.v3 round-trips
func preserveBlankLines(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	result := make([]string, 0, len(lines))

	inBlockScalar := false
	blockScalarIndent := 0
	pendingBlanks := 0

	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")

		// Buffer blank lines (except the trailing empty element from Split).
		if trimmed == "" && i < len(lines)-1 {
			pendingBlanks++
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Exit block scalar when indentation decreases.
		if inBlockScalar && indent <= blockScalarIndent {
			inBlockScalar = false
		}

		// Flush buffered blank lines: keep literal inside block scalars,
		// replace with markers otherwise.
		for range pendingBlanks {
			if inBlockScalar {
				result = append(result, "")
			} else {
				result = append(result, blankLineMarker)
			}
		}
		pendingBlanks = 0

		if !inBlockScalar && blockScalarRe.MatchString(trimmed) {
			inBlockScalar = true
			blockScalarIndent = indent
		}

		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n"))
}

// flushBlanks appends blank lines to result. When markers are present,
// it emits exactly that many blanks (dropping yaml.v3-added duplicates);
// otherwise it keeps all blanks as-is.
func flushBlanks(result []string, blanks, markers int) []string {
	n := blanks
	if markers > 0 {
		n = markers
	}
	for range n {
		result = append(result, "")
	}
	return result
}

// restoreBlankLines replaces marker comments back to blank lines.
func restoreBlankLines(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	result := make([]string, 0, len(lines))
	blanks := 0
	markers := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == blankLineMarker {
			markers++
			continue
		}
		if strings.TrimRight(line, " \t") == "" && i < len(lines)-1 {
			blanks++
			continue
		}
		result = flushBlanks(result, blanks, markers)
		blanks = 0
		markers = 0
		result = append(result, line)
	}
	result = flushBlanks(result, blanks, markers)

	out := strings.Join(result, "\n")

	// Safety net: replace any markers that survived due to unexpected reasons
	out = strings.ReplaceAll(out, blankLineMarker, "")

	return []byte(out)
}

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
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
	"go.yaml.in/yaml/v3"
)

// ApplyChangesToYAML generates YAML files for the given field changes. The second
// return value counts the changes left unapplied because their location in the
// YAML cannot be written.
func ApplyChangesToYAML(ctx context.Context, b *bundle.Bundle, fieldChanges []FieldChange) ([]FileChange, int, error) {
	originalFiles := make(map[string][]byte)
	modifiedFiles := make(map[string][]byte)
	fileFieldChanges := make(map[string][]FieldChange)
	skipped := 0

	for _, fieldChange := range fieldChanges {
		filePath := fieldChange.FilePath

		if _, exists := modifiedFiles[filePath]; !exists {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			originalFiles[filePath] = content
			modifiedFiles[filePath] = preserveBlankLines(content)
		}

		modifiedContent, err := applyChange(ctx, modifiedFiles[filePath], fieldChange)
		if err != nil {
			if errors.Is(err, errUnwritableParent) {
				log.Debugf(ctx, "config-remote-sync: skipping %s in %s: %v", fieldChange.WritePath, filePath, err)
				skipped++
				continue
			}
			return nil, 0, fmt.Errorf("failed to apply change to file %s for a field %s: %w", filePath, fieldChange.WritePath, err)
		}

		modifiedFiles[filePath] = modifiedContent
		fileFieldChanges[filePath] = append(fileFieldChanges[filePath], fieldChange)
	}

	var result []FileChange
	// Iterating the applied changes rather than modifiedFiles leaves out a file
	// whose every change was skipped: re-encoding it would report a modification
	// that was never made.
	for filePath, appliedChanges := range fileFieldChanges {
		// TODO: A good alternative approach is to remove parent nodes during the Resolve phase,
		// when all of their keys/items are removed, but this should be tested for edge cases.
		// In this case flow style will never appear because empty nodes are never serialized and we won't need clearAddedFlowStyle
		normalized, err := clearAddedFlowStyle(modifiedFiles[filePath], appliedChanges)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to normalize YAML style in %s: %w", filePath, err)
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

	return result, skipped, nil
}

// parentNode records what stands between a change and its destination: path is
// the change itself, ancestorPath the deepest node the walk reached, and kind how
// that node has to be materialized before the change can be applied.
type parentNode struct {
	path         yamlpatch.Path
	ancestorPath yamlpatch.Path
	kind         parentKind
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

			// Collect not-yet-writable parents for later retry. The patcher reports
			// a missing parent, an empty "key:" placeholder and a ${...} reference as
			// three differently-worded errors from a third-party package, so the
			// target node is inspected directly instead of matching on that text.
			if patchErr != nil {
				kind, ancestorPath, inspectErr := resolveParentNode(content, path)
				if inspectErr != nil {
					return nil, fmt.Errorf("failed to inspect parent of %s: %w", jsonPointer, inspectErr)
				}
				switch kind {
				case parentAbsent, parentNull, parentVariable:
					parentNodesToCreate = append(parentNodesToCreate, parentNode{path, ancestorPath, kind})
				default:
					// parentContainer means the parent is writable and the patch failed
					// for an unrelated reason; a plain scalar parent is a genuine error.
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

	// If all attempts failed because the parent is not a container yet, materialize
	// it and write the change into it.
	if !success && len(parentNodesToCreate) > 0 {
		for _, errInfo := range parentNodesToCreate {
			if errInfo.kind == parentVariable {
				// The node's value is owned by a variable: overwriting it destroys the
				// indirection for every target that shares this file.
				return nil, fmt.Errorf("%w at %s", errUnwritableParent, errInfo.ancestorPath.String())
			}

			nestedValue := buildNestedMaps(errInfo.path, errInfo.ancestorPath, fieldChange.Change.Value)

			// An empty "key:" is a null scalar rather than a mapping, so the
			// placeholder is replaced instead of added to.
			opType := yamlpatch.OperationAdd
			if errInfo.kind == parentNull {
				opType = yamlpatch.OperationReplace
			}

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			modifiedContent, patchErr := patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
				Type:  opType,
				Path:  errInfo.ancestorPath,
				Value: nestedValue,
			}})

			if patchErr == nil {
				content = modifiedContent
				firstErr = nil
				log.Debugf(ctx, "Created nested structure at %s", errInfo.ancestorPath.String())
				break
			}
			if firstErr == nil {
				firstErr = patchErr
			}
			log.Debugf(ctx, "Failed to create nested structure at %s: %v", errInfo.ancestorPath.String(), patchErr)
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

// isPathNotFoundError checks if error indicates the path itself does not exist.
func isPathNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist")
}

// errUnwritableParent reports that a change's parent node holds a variable
// reference, so the change has no location that can be written.
var errUnwritableParent = errors.New("parent value is a variable reference")

// parentKind classifies the node a change's parent path resolves to, which decides
// whether the change can be written and how.
type parentKind int

const (
	// parentContainer is a mapping or a sequence: the change can be applied as-is.
	parentContainer parentKind = iota
	// parentAbsent means the walk stopped before reaching the parent, so the
	// intermediate nodes have to be created.
	parentAbsent
	// parentNull is an empty "key:", which YAML parses as a null scalar rather than
	// as the empty mapping "key: {}" produces.
	parentNull
	// parentVariable is a scalar holding a ${...} reference.
	parentVariable
	// parentScalar is any other scalar, which nothing can be written underneath.
	parentScalar
)

// resolveParentNode walks path in content and reports the kind of the deepest node
// the walk reached along with that node's path, mirroring how the patcher resolves
// the same path: the last path segment is the key being written, so only its
// ancestors are traversed, and an absent node is reported at the path the patcher
// names in its "parent path %s does not exist" error.
func resolveParentNode(content []byte, path yamlpatch.Path) (parentKind, yamlpatch.Path, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return parentContainer, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	current := &doc
	for i := range path {
		if current == nil {
			return parentAbsent, path[:i], nil
		}
		if kind := nodeKind(current); kind != parentContainer {
			return kind, path[:i], nil
		}
		if i == len(path)-1 {
			break
		}
		current = childNode(current, path[i])
	}

	return parentContainer, nil, nil
}

// nodeKind classifies a single node. Aliases are dereferenced because the patcher
// does the same, so a change addressed through a YAML anchor stays writable.
func nodeKind(node *yaml.Node) parentKind {
	node = derefAlias(node)

	switch node.Kind {
	case yaml.DocumentNode:
		// The patcher refuses a document that does not hold exactly one value.
		if len(node.Content) != 1 {
			return parentScalar
		}
		return parentContainer
	case yaml.MappingNode, yaml.SequenceNode:
		return parentContainer
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			return parentNull
		}
		if dynvar.ContainsVariableReference(node.Value) {
			return parentVariable
		}
		return parentScalar
	default:
		// Content that did not parse into a node at all (an empty file) is no more
		// writable than a scalar; aliases are already dereferenced above.
		return parentScalar
	}
}

// childNode returns the node stored under key, or nil when the container has no
// such child. It follows the lookup rules of the patcher's containers, including
// reporting an empty document as an absent child rather than a null scalar.
func childNode(node *yaml.Node, key string) *yaml.Node {
	node = derefAlias(node)

	switch node.Kind {
	case yaml.DocumentNode:
		if root := node.Content[0]; root.Kind != yaml.ScalarNode || root.Tag != "!!null" {
			return root
		}
	case yaml.MappingNode:
		// Content is [key1, val1, key2, val2, ...].
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return node.Content[i+1]
			}
		}
	case yaml.SequenceNode:
		// The "-" append marker and an out-of-range index both address an element
		// that does not exist yet.
		if idx, err := strconv.Atoi(key); err == nil && idx >= 0 && idx < len(node.Content) {
			return node.Content[idx]
		}
	default:
		// Only containers are walked into, so nothing else reaches this.
	}

	return nil
}

func derefAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
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
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
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

		if key, ok := n.StringKey(); ok {
			if current.Kind != yaml.MappingNode {
				return
			}
			found := false
			// current.Content: [key1, val1, key2, val2, ...]
			for i := 0; i+1 < len(current.Content); i += 2 {
				if current.Content[i].Value == key {
					current = current.Content[i+1]
					found = true
					break
				}
			}
			if !found {
				return
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			if current.Kind != yaml.SequenceNode || idx < 0 || idx >= len(current.Content) {
				return
			}
			current = current.Content[idx]
			continue
		}

		return
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

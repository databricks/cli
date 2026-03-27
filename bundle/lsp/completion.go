package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

const (
	completionKindVariable = 6  // LSP CompletionItemKind.Variable
	completionKindField    = 5  // LSP CompletionItemKind.Field
	completionKindModule   = 9  // LSP CompletionItemKind.Module
	completionKindValue    = 12 // LSP CompletionItemKind.Value
)

// CompletionContext holds the parsed state of a partial interpolation at the cursor.
type CompletionContext struct {
	// Start is the character offset of the `${` opening in the line.
	Start int
	// PartialPath is the text between `${` and the cursor (e.g., "var.clust").
	PartialPath string
}

// FindCompletionContext locates a partial `${...` interpolation at the cursor position.
// Returns the context and true if the cursor is inside an incomplete interpolation.
func FindCompletionContext(lines []string, pos Position) (CompletionContext, bool) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return CompletionContext{}, false
	}

	line := lines[pos.Line]
	if pos.Character > len(line) {
		return CompletionContext{}, false
	}

	// Look backwards from cursor for the nearest unmatched "${".
	textBefore := line[:pos.Character]
	dollarBrace := strings.LastIndex(textBefore, "${")
	if dollarBrace < 0 {
		return CompletionContext{}, false
	}

	// Make sure there's no closing "}" between "${" and cursor.
	afterOpen := textBefore[dollarBrace+2:]
	if strings.Contains(afterOpen, "}") {
		return CompletionContext{}, false
	}

	return CompletionContext{
		Start:       dollarBrace,
		PartialPath: afterOpen,
	}, true
}

// CompleteInterpolation returns completion items for a partial interpolation path.
// editRange is the range to replace (from after "${" to cursor position).
func CompleteInterpolation(tree dyn.Value, partial string, editRange *Range) []CompletionItem {
	if !tree.IsValid() {
		return nil
	}

	// Rewrite "var" / "var." shorthand to "variables" / "variables." for tree lookup.
	lookupPath := partial
	isVarShorthand := strings.HasPrefix(partial, "var.") || partial == "var"
	if partial == "var" {
		lookupPath = "variables."
	} else if isVarShorthand {
		lookupPath = "variables." + strings.TrimPrefix(partial, "var.")
	}

	// Check if the user is typing an index like "foo.bar[" or "foo.bar[0".
	// In that case, navigate to "foo.bar" and suggest indices.
	if basePath, ok := parsePartialIndex(lookupPath); ok {
		return completeSequenceIndices(tree, basePath, partial, isVarShorthand, editRange)
	}

	// Split into parent path and the prefix the user is currently typing.
	parentPath, prefix := splitPartialPath(lookupPath)

	// Navigate to the parent node in the tree.
	parent := tree
	parentFound := true
	if parentPath != "" {
		p, err := dyn.NewPathFromString(parentPath)
		if err != nil {
			parentFound = false
		} else {
			parent, err = dyn.GetByPath(tree, p)
			if err != nil {
				parentFound = false
			}
		}
	}

	// If parent is a sequence, suggest indexed access (e.g., path[0], path[1]).
	if parentFound && parent.Kind() == dyn.KindSequence {
		return completeSequenceIndices(tree, parentPath, partial, isVarShorthand, editRange)
	}

	// Collect child keys from the parent map.
	var items []CompletionItem
	if parentFound && parent.Kind() == dyn.KindMap {
		m, ok := parent.AsMap()
		if ok {
			for _, pair := range m.Pairs() {
				key := pair.Key.MustString()
				if prefix != "" && !strings.HasPrefix(key, prefix) {
					continue
				}

				child := pair.Value

				// Expand sequences inline: instead of showing "tasks" (list),
				// show "tasks[0]", "tasks[1]", etc. directly.
				if child.Kind() == dyn.KindSequence {
					seq, ok := child.AsSequence()
					if ok {
						basePath := buildDisplayPath(parentPath, key, isVarShorthand)
						for i, elem := range seq {
							indexedPath := fmt.Sprintf("%s[%d]", basePath, i)
							// Use dot-separated filter text so VSCode's completion
							// engine can match it (brackets confuse the fuzzy matcher).
							filterPath := fmt.Sprintf("%s.%d", basePath, i)
							kind, detail := classifyValue(elem)
							item := CompletionItem{
								Label:      indexedPath,
								Kind:       kind,
								Detail:     detail,
								FilterText: filterPath,
							}
							applyTextEdit(&item, indexedPath, editRange)
							items = append(items, item)
						}
					}
					continue
				}

				displayPath := buildDisplayPath(parentPath, key, isVarShorthand)
				kind, detail := classifyValue(child)

				item := CompletionItem{
					Label:      displayPath,
					Kind:       kind,
					Detail:     detail,
					FilterText: displayPath,
				}
				applyTextEdit(&item, displayPath, editRange)
				items = append(items, item)
			}
		}
	}

	// Merge computed keys that match the partial path.
	items = mergeComputedItems(items, partial, editRange)

	if len(items) == 0 {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

// mergeComputedItems appends computed completion items that don't duplicate existing tree items.
func mergeComputedItems(items []CompletionItem, partial string, editRange *Range) []CompletionItem {
	computed := computedCompletions(partial, editRange)
	existing := make(map[string]bool, len(items))
	for _, it := range items {
		existing[it.Label] = true
	}
	for _, c := range computed {
		if !existing[c.Label] {
			items = append(items, c)
		}
	}
	return items
}

// completeSequenceIndices suggests [0], [1], ... for a sequence node.
func completeSequenceIndices(tree dyn.Value, seqPath, partial string, isVarShorthand bool, editRange *Range) []CompletionItem {
	node := tree
	if seqPath != "" {
		p, err := dyn.NewPathFromString(seqPath)
		if err != nil {
			return nil
		}
		node, err = dyn.GetByPath(tree, p)
		if err != nil {
			return nil
		}
	}

	seq, ok := node.AsSequence()
	if !ok {
		return nil
	}

	// Build the display prefix (rewrite variables back to var shorthand).
	displayPrefix := seqPath
	if isVarShorthand {
		displayPrefix = rewriteToVarShorthand(seqPath)
	}

	var items []CompletionItem
	for i, elem := range seq {
		displayPath := fmt.Sprintf("%s[%d]", displayPrefix, i)
		// Use dot-separated filter text so VSCode's completion
		// engine can match it (brackets confuse the fuzzy matcher).
		filterPath := fmt.Sprintf("%s.%d", displayPrefix, i)
		kind, detail := classifyValue(elem)

		item := CompletionItem{
			Label:      displayPath,
			Kind:       kind,
			Detail:     detail,
			FilterText: filterPath,
		}
		applyTextEdit(&item, displayPath, editRange)
		items = append(items, item)
	}
	return items
}

// parsePartialIndex checks if the path ends with a partial index like "foo.bar[" or "foo.bar[1".
// Returns the base path ("foo.bar") and true if so.
func parsePartialIndex(path string) (string, bool) {
	bracketIdx := strings.LastIndex(path, "[")
	if bracketIdx < 0 {
		return "", false
	}

	// Only match if "[" is the last bracket and there's no closing "]" after it.
	after := path[bracketIdx:]
	if strings.Contains(after, "]") {
		return "", false
	}

	return path[:bracketIdx], true
}

// TopLevelCompletions returns completions for when the user just typed "${" with no path yet.
func TopLevelCompletions(tree dyn.Value, editRange *Range) []CompletionItem {
	items := CompleteInterpolation(tree, "", editRange)

	// Add "var" shorthand if "variables" exists in the tree.
	vars := tree.Get("variables")
	if vars.Kind() == dyn.KindMap {
		item := CompletionItem{
			Label:      "var",
			Kind:       completionKindVariable,
			Detail:     "variable shorthand",
			FilterText: "var",
		}
		applyTextEdit(&item, "var", editRange)
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

// splitPartialPath splits "resources.jobs.my_j" into parent="resources.jobs" and prefix="my_j".
// If there's no dot, parent="" and prefix is the whole string.
// Handles paths with index suffixes like "foo.bar[0].baz" correctly.
func splitPartialPath(partial string) (parent, prefix string) {
	idx := strings.LastIndex(partial, ".")
	if idx < 0 {
		return "", partial
	}
	return partial[:idx], partial[idx+1:]
}

// buildDisplayPath constructs the full display path, handling var shorthand.
func buildDisplayPath(parentPath, key string, isVarShorthand bool) string {
	if parentPath == "" {
		return key
	}
	if isVarShorthand {
		return rewriteToVarShorthand(parentPath) + "." + key
	}
	return parentPath + "." + key
}

// rewriteToVarShorthand rewrites a "variables..." path back to "var..." for display.
func rewriteToVarShorthand(path string) string {
	if path == "variables" {
		return "var"
	}
	if strings.HasPrefix(path, "variables.") {
		return "var." + strings.TrimPrefix(path, "variables.")
	}
	if strings.HasPrefix(path, "variables[") {
		return "var" + strings.TrimPrefix(path, "variables")
	}
	return path
}

// classifyValue returns the completion kind and detail string for a dyn.Value.
func classifyValue(v dyn.Value) (int, string) {
	switch v.Kind() {
	case dyn.KindMap:
		return completionKindModule, "map"
	case dyn.KindSequence:
		return completionKindModule, "list"
	case dyn.KindString:
		if s, ok := v.AsString(); ok {
			return completionKindValue, s
		}
	case dyn.KindBool:
		if b, ok := v.AsBool(); ok {
			if b {
				return completionKindValue, "true"
			}
			return completionKindValue, "false"
		}
	case dyn.KindInt, dyn.KindFloat:
		return completionKindValue, "number"
	case dyn.KindInvalid, dyn.KindNil, dyn.KindTime:
		// These kinds are not expected in bundle YAML but are handled
		// for exhaustiveness.
	}
	return completionKindField, ""
}

// computedKeys are keys that exist at runtime but are not present in YAML config files.
// Users can reference these in ${...} interpolation expressions.
var computedKeys = []string{
	"bundle.target",
	"bundle.environment",
	"bundle.git.branch",
	"bundle.git.origin_url",
	"bundle.git.commit",
	"bundle.git.actual_branch",
	"bundle.git.bundle_root_path",
	"workspace.current_user.short_name",
	"workspace.current_user.user_name",
	"workspace.root_path",
	"workspace.file_path",
	"workspace.resource_path",
	"workspace.artifact_path",
	"workspace.state_path",
}

// computedCompletions returns completion items for computed keys matching the partial path prefix.
func computedCompletions(partial string, editRange *Range) []CompletionItem {
	var items []CompletionItem
	for _, key := range computedKeys {
		if partial != "" && !strings.HasPrefix(key, partial) {
			continue
		}
		// Only show exact-depth children: if partial is "bundle.", show "bundle.target"
		// but not "bundle.git.commit" (that requires "bundle.git." first).
		suffix := strings.TrimPrefix(key, partial)
		if dotIdx := strings.Index(suffix, "."); dotIdx >= 0 {
			// The computed key has more depth; show the intermediate segment instead.
			// e.g., for partial="bundle." and key="bundle.git.commit", show "bundle.git".
			intermediate := partial + suffix[:dotIdx]
			// Check if we already added this intermediate.
			found := false
			for _, it := range items {
				if it.Label == intermediate {
					found = true
					break
				}
			}
			if !found {
				item := CompletionItem{
					Label:      intermediate,
					Kind:       completionKindModule,
					Detail:     "computed",
					FilterText: intermediate,
				}
				applyTextEdit(&item, intermediate, editRange)
				items = append(items, item)
			}
			continue
		}

		item := CompletionItem{
			Label:      key,
			Kind:       completionKindVariable,
			Detail:     "computed",
			FilterText: key,
		}
		applyTextEdit(&item, key, editRange)
		items = append(items, item)
	}
	return items
}

// applyTextEdit sets either a TextEdit or InsertText on the completion item.
func applyTextEdit(item *CompletionItem, text string, editRange *Range) {
	if editRange != nil {
		item.TextEdit = &TextEdit{
			Range:   *editRange,
			NewText: text,
		}
	} else {
		item.InsertText = text
	}
}

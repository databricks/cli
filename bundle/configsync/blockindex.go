package configsync

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
)

// A sequence field of a resource may be defined in more than one physical YAML
// region: the top-level resources.<type>.<name>.<field> block and the
// targets.<target>.resources.<type>.<name>.<field> override block, either of
// which may live in its own included file. Loading merges those regions into one
// sequence, and keyed sequences are also sorted by key, so an index into the
// merged sequence does not address any single region. Write-back has to map a
// merged position back to a region and to the position inside that region.
type sourceBlock struct {
	// prefix is empty for a top-level block and "targets.<target>" for a target
	// override block. Both may occur in the same file, so the file alone does not
	// identify a block.
	prefix string
	file   string
}

// blockResolver answers which physical block a merged value came from.
//
// It works by location: a merged value keeps the locations of the entries it was
// loaded from, and merging accumulates them (see libs/dyn/merge), so a value
// assembled from two blocks reports a location in each. Selecting a target folds
// its overrides into the resources tree and drops the targets subtree, so the
// blocks are recovered by parsing the contributing files again, where the two
// regions are still separate.
type blockResolver struct {
	// blocks holds one parsed file per contributing file, keyed by block, so a
	// path relative to a block can be looked up inside it.
	blocks     map[sourceBlock]dyn.Value
	target     string
	byLocation map[dyn.Location]sourceBlock
}

// newBlockResolver builds the location -> block mapping for the bundle's
// resources. It returns nil when the source cannot be re-read, in which case
// callers keep their existing single-block behaviour.
//
// Only the files the merged configuration actually references are read, and they
// are parsed directly rather than reloaded through the mutator pipeline: the
// pipeline resolves includes, reports through logdiag and executes the bundle's
// preinit script, none of which belongs to reading back a source location.
func newBlockResolver(ctx context.Context, b *bundle.Bundle) *blockResolver {
	root := b.Config.Value()
	if !root.IsValid() {
		return nil
	}

	r := &blockResolver{
		blocks:     make(map[sourceBlock]dyn.Value),
		target:     b.Config.Bundle.Target,
		byLocation: make(map[dyn.Location]sourceBlock),
	}

	for _, file := range slices.Sorted(maps.Keys(referencedFiles(root))) {
		contents, err := os.ReadFile(file)
		if err != nil {
			log.Debugf(ctx, "config-remote-sync: cannot read %s, treating its sequences as unsplit: %v", file, err)
			continue
		}
		parsed, diags := config.LoadFromBytes(file, contents)
		if diags.HasError() {
			log.Debugf(ctx, "config-remote-sync: cannot parse %s, treating its sequences as unsplit: %v", file, diags.Error())
			continue
		}
		r.indexRegion(parsed.Value(), sourceBlock{file: file})
		if r.target != "" {
			r.indexRegion(parsed.Value(), sourceBlock{prefix: "targets." + r.target, file: file})
		}
	}

	if len(r.byLocation) == 0 {
		return nil
	}
	return r
}

// referencedFiles returns the files the merged configuration was loaded from.
func referencedFiles(root dyn.Value) map[string]struct{} {
	files := map[string]struct{}{}
	_ = dyn.WalkReadOnly(root, func(_ dyn.Path, v dyn.Value) error {
		for _, location := range v.Locations() {
			if location.File != "" {
				files[location.File] = struct{}{}
			}
		}
		return nil
	})
	return files
}

// indexRegion records every location under one region's resources subtree of a
// parsed file, and keeps the parsed file so paths can be resolved inside it.
func (r *blockResolver) indexRegion(parsed dyn.Value, block sourceBlock) {
	subtree, err := dyn.GetByPath(parsed, r.regionPath(block.prefix, dyn.NewPath(dyn.Key("resources"))))
	if err != nil {
		return
	}
	r.blocks[block] = parsed
	_ = dyn.WalkReadOnly(subtree, func(_ dyn.Path, v dyn.Value) error {
		for _, location := range v.Locations() {
			// A file only carries locations of its own, so a location seen here
			// belongs to this block. First writer wins: an outer node accumulates
			// its children's locations, but the innermost node that owns a location
			// is the one walked last.
			if location.File != block.file {
				continue
			}
			if _, ok := r.byLocation[location]; !ok {
				r.byLocation[location] = block
			}
		}
		return nil
	})
}

// regionPath prefixes a resources-relative path with the region it belongs to.
func (r *blockResolver) regionPath(prefix string, path dyn.Path) dyn.Path {
	if prefix == "" {
		return path
	}
	return append(dyn.NewPath(dyn.Key("targets"), dyn.Key(r.target)), path...)
}

// sortedBlocks lists the known blocks with the top-level ones first and a total
// order within each scope, so a choice between blocks never depends on map or
// location iteration order.
func (r *blockResolver) sortedBlocks() []sourceBlock {
	blocks := slices.Collect(maps.Keys(r.blocks))
	slices.SortFunc(blocks, compareBlocks)
	return blocks
}

// compareBlocks orders top-level blocks before target blocks, then by file.
func compareBlocks(a, b sourceBlock) int {
	if (a.prefix == "") != (b.prefix == "") {
		if a.prefix == "" {
			return -1
		}
		return 1
	}
	return cmp.Or(cmp.Compare(a.prefix, b.prefix), cmp.Compare(a.file, b.file))
}

// blocksOf returns the distinct blocks that contributed to value, sorted with the
// top-level block first. More than one result means the value is assembled from
// several regions and has no single source location.
func (r *blockResolver) blocksOf(value dyn.Value) []sourceBlock {
	var blocks []sourceBlock
	for _, location := range value.Locations() {
		block, ok := r.byLocation[location]
		if !ok {
			continue
		}
		if !slices.Contains(blocks, block) {
			blocks = append(blocks, block)
		}
	}
	// Order is total and independent of how locations happened to accumulate, so
	// callers that prefer the top-level block get a deterministic answer.
	slices.SortFunc(blocks, compareBlocks)
	return blocks
}

// winningBlock returns the block whose definition of value is the one the merged
// configuration ended up with.
//
// A scalar defined in several blocks is not ambiguous: merging keeps the incoming
// value and records its location first (see mergePrimitive in libs/dyn/merge), so
// the deployed value came from whichever block owns Locations()[0]. Writing the
// remote change anywhere else would leave the effective value unchanged.
func (r *blockResolver) winningBlock(value dyn.Value) (sourceBlock, bool) {
	for _, location := range value.Locations() {
		if block, ok := r.byLocation[location]; ok {
			return block, true
		}
	}
	return sourceBlock{}, false
}

// localIndex returns the position of element inside the sequence that block
// defines at sequencePath, where sequencePath is relative to the block.
func (r *blockResolver) localIndex(block sourceBlock, sequencePath dyn.Path, element dyn.Value) (int, bool) {
	parsed, ok := r.blocks[block]
	if !ok {
		return 0, false
	}
	sequence, err := dyn.GetByPath(parsed, r.regionPath(block.prefix, sequencePath))
	if err != nil {
		return 0, false
	}
	entries, ok := sequence.AsSequence()
	if !ok {
		return 0, false
	}

	locations := make(map[dyn.Location]struct{}, len(element.Locations()))
	for _, location := range element.Locations() {
		locations[location] = struct{}{}
	}

	// A region's sequence is the concatenation of the entries contributed by each
	// included file, but the write targets one file, so the index has to be
	// counted among that file's entries only.
	local := 0
	for _, entry := range entries {
		entryFile := entry.Location().File
		if entryFile != block.file {
			continue
		}
		for _, location := range entry.Locations() {
			if _, ok := locations[location]; ok {
				return local, true
			}
		}
		local++
	}
	return 0, false
}

// errAmbiguousBlock reports a change that cannot be attributed to one physical
// block. Leaving such a change unapplied is preferable to writing it to a guessed
// location, since the sync runs unattended and a later run can retry.
var errAmbiguousBlock = errors.New("change cannot be attributed to a single source block")

// route maps a change resolved against the merged configuration onto the physical
// block that owns it, rewriting merged sequence indices into block-local ones.
//
// The element that the change addresses decides the block: an element defined in
// exactly one block is written there. When the addressed element is assembled from
// several blocks, the leaf field decides instead, because merging records the
// location of each field separately. If neither identifies a single block the
// change is ambiguous and is reported as such.
func (r *blockResolver) route(change resolvedChange) (sourceBlock, *structpath.PatternNode, error) {
	block, err := r.blockFor(change)
	if err != nil {
		return sourceBlock{}, nil, err
	}

	path, err := r.localize(block, change)
	if err != nil {
		return sourceBlock{}, nil, err
	}
	return block, path, nil
}

// blockFor picks the block a change belongs to.
func (r *blockResolver) blockFor(change resolvedChange) (sourceBlock, error) {
	// A new element has no source of its own; it is placed relative to the
	// sequence that receives it.
	for _, step := range change.steps {
		if step.newElement {
			return r.blockForNewElement(change)
		}
	}

	if len(change.steps) > 0 {
		last := change.steps[len(change.steps)-1]
		blocks := r.blocksOf(last.element)
		switch len(blocks) {
		case 1:
			return blocks[0], nil
		case 0:
			return sourceBlock{}, fmt.Errorf("%w: no source location for the addressed element", errAmbiguousBlock)
		}

		// The element is assembled from several blocks, but a field of it is not:
		// it is written in one block, or written in both and one of them wins. A
		// field-level change therefore routes by the field's own location.
		if change.leaf.IsValid() {
			if block, ok := r.winningBlock(change.leaf); ok {
				return block, nil
			}
			return sourceBlock{}, fmt.Errorf("%w: element is defined in %d blocks and the field has no source location", errAmbiguousBlock, len(blocks))
		}

		// A field that exists in no block is new. It has no location to route by,
		// but it still needs a defined destination, so it goes to the block that
		// declares the resource rather than a target-specific scope.
		if len(change.path.AsSlice()) > last.component+1 {
			for _, block := range blocks {
				if block.prefix == "" {
					return block, nil
				}
			}
			return blocks[0], nil
		}

		// The change addresses the element itself, not one of its fields. Removing
		// or recreating it cannot be expressed against a single block.
		return sourceBlock{}, fmt.Errorf("%w: element is defined in %d blocks", errAmbiguousBlock, len(blocks))
	}

	// No sequence on the path: route by the leaf's own location.
	if change.leaf.IsValid() {
		if block, ok := r.winningBlock(change.leaf); ok {
			return block, nil
		}
	}
	return sourceBlock{}, fmt.Errorf("%w: no source location for the change", errAmbiguousBlock)
}

// blockForNewElement chooses where an element that exists in no source block
// should be written: the block that defines the sequence, or when several do, the
// block that declares the resource. Adding to the resource's own block keeps a new
// element out of a target-specific scope the user did not ask for.
func (r *blockResolver) blockForNewElement(change resolvedChange) (sourceBlock, error) {
	step := change.steps[len(change.steps)-1]

	// A nested sequence is reached through the enclosing elements, which are
	// already placed: route by the innermost one that exists in the source, since
	// a new element belongs in the same block as its parent.
	for i := len(change.steps) - 2; i >= 0; i-- {
		enclosing := change.steps[i]
		if enclosing.newElement {
			continue
		}
		if blocks := r.blocksOf(enclosing.element); len(blocks) == 1 {
			return blocks[0], nil
		}
	}

	var blocks []sourceBlock
	for _, block := range r.sortedBlocks() {
		sequence, err := dyn.GetByPath(r.blocks[block], r.regionPath(block.prefix, step.sequencePath))
		if err != nil {
			continue
		}
		for _, location := range sequence.Locations() {
			if location.File == block.file && !slices.Contains(blocks, block) {
				blocks = append(blocks, block)
			}
		}
	}

	switch len(blocks) {
	case 0:
		return sourceBlock{}, fmt.Errorf("%w: no block defines the sequence receiving the new element", errAmbiguousBlock)
	case 1:
		return blocks[0], nil
	}

	// blocksOf orders the top-level block first, and the resource is declared
	// there whenever a top-level block exists at all.
	for _, block := range blocks {
		if block.prefix == "" {
			return block, nil
		}
	}
	return blocks[0], nil
}

// localize rewrites the merged sequence indices in a change's path into indices
// inside block.
func (r *blockResolver) localize(block sourceBlock, change resolvedChange) (*structpath.PatternNode, error) {
	nodes := change.path.AsSlice()
	local := make([]int, len(change.steps))
	for i, step := range change.steps {
		if step.newElement {
			// Keeps the [*] placeholder; the patcher appends to the block's
			// sequence.
			local[i] = -1
			continue
		}
		// A nested sequence is reached through the outer sequences on the path,
		// whose indices refer to the merged view. Rewrite them to the block's own
		// indices, otherwise the lookup addresses the wrong parent element.
		sequencePath := slices.Clone(step.sequencePath)
		for j := range i {
			outer := change.steps[j]
			if local[j] < 0 || len(outer.sequencePath) >= len(sequencePath) {
				continue
			}
			sequencePath[len(outer.sequencePath)] = dyn.Index(local[j])
		}

		index, ok := r.localIndex(block, sequencePath, step.element)
		if !ok {
			return nil, fmt.Errorf("%w: element has no position in %s", errAmbiguousBlock, block.file)
		}
		local[i] = index
	}

	var result *structpath.PatternNode
	next := 0
	for component, node := range nodes {
		if next < len(change.steps) && change.steps[next].component == component {
			if local[next] < 0 {
				result = structpath.NewPatternBracketStar(result)
			} else {
				result = structpath.NewPatternIndex(result, local[next])
			}
			next++
			continue
		}
		key, ok := node.StringKey()
		if !ok {
			return nil, fmt.Errorf("%w: unsupported path component in %s", errAmbiguousBlock, change.path.String())
		}
		result = structpath.NewPatternStringKey(result, key)
	}
	return result, nil
}

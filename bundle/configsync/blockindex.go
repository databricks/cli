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

// Write-back has to turn a position in the merged configuration into a position in a
// file, and the two do not correspond.
//
// A sequence field of a resource may be defined in two physical regions: the top-level
// resources.<type>.<name>.<field> block and the targets.<target>.resources... override
// block, either of which may live in its own included file. Loading concatenates those
// regions into one sequence and sorts keyed ones by key, so a merged index addresses
// no single region. Every index therefore exists in one of two spaces -- merged, or
// local to one block -- and each value in this file belongs to exactly one of them;
// crossing them silently addresses a different element.
//
// A block is (kind, file), because both kinds may occur in one file and one target's
// override may span several files, so neither identifies a block alone.
type sourceBlock struct {
	// override is false for the top-level block and true for the selected target's
	// override block.
	override bool
	file     string
}

// scopeKey identifies this block for index bookkeeping, so operations on one block
// cannot shift positions recorded for another.
func (b sourceBlock) scopeKey() string {
	if b.override {
		return "override\x00" + b.file + "\x00"
	}
	return "toplevel\x00" + b.file + "\x00"
}

// blockResolver answers which physical block a merged value came from, by location:
// merging accumulates them (libs/dyn/merge), so a value assembled from two blocks
// reports a location in each. Selecting a target folds its overrides into resources and
// drops the targets subtree, so the blocks are recovered by parsing the files again.
type blockResolver struct {
	// blocks holds one parsed file per contributing file, keyed by block, so a
	// path relative to a block can be looked up inside it.
	blocks     map[sourceBlock]dyn.Value
	target     string
	byLocation map[dyn.Location]sourceBlock
}

// newBlockResolver builds the location -> block mapping for the bundle's resources,
// returning nil when the source cannot be re-read so callers keep their existing
// single-block behaviour.
//
// The contributing files are parsed directly rather than reloaded through the mutator
// pipeline, which would run the bundle's preinit script a second time.
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

	sourceFiles := slices.Sorted(maps.Keys(referencedFiles(root)))
	for _, file := range sourceFiles {
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
		r.registerBlock(parsed.Value(), sourceBlock{file: file})
		if r.target != "" {
			r.registerBlock(parsed.Value(), sourceBlock{override: true, file: file})
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

// registerBlock records that block exists and maps every location under its
// resources subtree back to it, so a merged value can later be traced to the
// region it was written in. Does nothing when the file has no such region, which
// is why r.blocks doubles as the set of blocks that exist.
func (r *blockResolver) registerBlock(parsed dyn.Value, block sourceBlock) {
	subtree, err := dyn.GetByPath(parsed, r.regionPath(block, dyn.NewPath(dyn.Key("resources"))))
	if err != nil {
		return
	}

	// Keep the parsed file: resolving a path inside this block needs the tree it
	// came from, and the entry also marks the block as present for sortedBlocks.
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
func (r *blockResolver) regionPath(block sourceBlock, path dyn.Path) dyn.Path {
	if !block.override {
		return path
	}
	return append(dyn.NewPath(dyn.Key("targets"), dyn.Key(r.target)), path...)
}

// candidatePath renders a resolved path the way the patch layer addresses it,
// which for an override block includes the targets.<target> prefix.
//
// The prefix is built as path nodes rather than concatenated as text: a target name
// may contain a dot ("dev.eu" is a legal name), and the result is parsed back into
// segments, so text would split one name into two keys and address the wrong node.
func (r *blockResolver) candidatePath(block sourceBlock, path *structpath.PatternNode) string {
	if !block.override {
		return path.String()
	}
	return targetPrefixedPath(r.target, path)
}

// targetPrefixedPath prefixes path with targets.<target>, quoting the target name
// where necessary so the result parses back into the same segments.
func targetPrefixedPath(target string, path *structpath.PatternNode) string {
	prefixed := structpath.NewPatternStringKey(nil, "targets")
	prefixed = structpath.NewPatternStringKey(prefixed, target)
	for _, node := range path.AsSlice() {
		if key, ok := node.StringKey(); ok {
			prefixed = structpath.NewPatternStringKey(prefixed, key)
		} else if index, ok := node.Index(); ok {
			prefixed = structpath.NewPatternIndex(prefixed, index)
		} else {
			prefixed = structpath.NewPatternBracketStar(prefixed)
		}
	}
	return prefixed.String()
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
	if a.override != b.override {
		if !a.override {
			return -1
		}
		return 1
	}
	return cmp.Compare(a.file, b.file)
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
	// Total order, so declaringBlock's choice never depends on how locations
	// happened to accumulate.
	slices.SortFunc(blocks, compareBlocks)
	return blocks
}

// winningBlock returns the block whose definition the merged value took.
//
// Locations accumulate in merge order, so the first one that maps to a block is the
// winner and writing any other copy would leave the effective value unchanged. Load
// order is the only thing that distinguishes two blocks in the same scope.
func (r *blockResolver) winningBlock(value dyn.Value) (sourceBlock, bool) {
	for _, location := range value.Locations() {
		if block, ok := r.byLocation[location]; ok {
			return block, true
		}
	}
	return sourceBlock{}, false
}

// indexWithinBlock returns the position of element inside block, where sequencePath is
// relative to the block. A block is one parsed file, so the sequence read here holds
// only that file's entries and a plain index into it is block-local.
func (r *blockResolver) indexWithinBlock(block sourceBlock, sequencePath dyn.Path, element dyn.Value) (int, bool) {
	parsed, ok := r.blocks[block]
	if !ok {
		return 0, false
	}
	sequence, err := dyn.GetByPath(parsed, r.regionPath(block, sequencePath))
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

	for local, entry := range entries {
		for _, location := range entry.Locations() {
			if _, ok := locations[location]; ok {
				return local, true
			}
		}
	}
	return 0, false
}

// errAmbiguousBlock reports a change that cannot be attributed to one physical
// block. Leaving such a change unapplied is preferable to writing it to a guessed
// location, since the sync runs unattended and a later run can retry.
var errAmbiguousBlock = errors.New("change cannot be attributed to a single source block")

// singleDestination maps a change onto the one block that owns it.
//
// The element the change addresses decides the block: an element defined in
// exactly one block is written there. When the addressed element is assembled from
// several blocks, the leaf field decides instead, because merging records the
// location of each field separately. If neither identifies a single block the
// change is ambiguous and is reported as such.
func (r *blockResolver) singleDestination(change resolvedChange) (routeDestination, error) {
	block, err := r.blockFor(change)
	if err != nil {
		return routeDestination{}, err
	}

	path, err := r.pathWithinBlock(block, change)
	if err != nil {
		return routeDestination{}, err
	}
	return routeDestination{block: &block, path: path}, nil
}

// routeDestination is one physical place a change has to be written. block is nil when
// the change was not routed to a block, i.e. it is written at its resolved path in
// whichever file the resource was found in.
type routeDestination struct {
	block *sourceBlock
	path  *structpath.PatternNode
}

// scopeKey identifies the destination for index bookkeeping, so operations on one
// block cannot shift positions recorded for another. An unrouted destination shares
// one scope, which is the behaviour before blocks were known.
func (d routeDestination) scopeKey() string {
	if d.block == nil {
		return ""
	}
	return d.block.scopeKey()
}

// routeDestinations returns every physical place a change has to be written: one for an
// edit, since only the definition that wins the merge decides the deployed value, and
// one per definition for a removal, since the value is gone only once every copy is.
//
// Callers only route a change that addresses something inside a sequence.
func (r *blockResolver) routeDestinations(change resolvedChange) ([]routeDestination, error) {
	// The value whose definitions have to be reached: the element itself when the
	// change addresses one, otherwise the field being removed.
	var target dyn.Value
	if addressesWholeElement(change) {
		target = change.steps[len(change.steps)-1].element
	} else if change.operation == OperationRemove {
		target = change.leaf
	}

	if !target.IsValid() {
		destination, err := r.singleDestination(change)
		if err != nil {
			return nil, err
		}
		return []routeDestination{destination}, nil
	}

	blocks := r.blocksOf(target)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("%w: no source location for the value being changed", errAmbiguousBlock)
	}

	destinations := make([]routeDestination, 0, len(blocks))
	for _, block := range blocks {
		path, err := r.pathWithinBlock(block, change)
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, routeDestination{block: &block, path: path})
	}
	return destinations, nil
}

// addressesWholeElement reports whether the change targets an existing sequence
// element itself rather than something inside it. Only such a change can need more
// than one destination, since only it can span the blocks the element is built
// from. A new element is excluded: it exists in no block yet, so it has no parts to
// span and is placed by blockForNewElement instead.
func addressesWholeElement(change resolvedChange) bool {
	if len(change.steps) == 0 {
		return false
	}
	for _, step := range change.steps {
		if step.newElement {
			return false
		}
	}
	last := change.steps[len(change.steps)-1]
	return len(change.path.AsSlice()) == last.component+1
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

	element := change.steps[len(change.steps)-1].element
	blocks := r.blocksOf(element)
	if len(blocks) == 0 {
		return sourceBlock{}, fmt.Errorf("%w: no source location for the addressed element", errAmbiguousBlock)
	}
	if len(blocks) == 1 {
		return blocks[0], nil
	}

	// The element is assembled from several blocks. A field of it is not: it is
	// written in one block, or written in both and one of them wins, so a
	// field-level change routes by the field's own location.
	if block, ok := r.winningBlock(change.leaf); ok {
		return block, nil
	}

	// The field has no source location: either it is absent from the config, or a
	// mutator inserted it (e.g. OverrideCompute setting existing_cluster_id), which
	// builds a fresh value carrying no location. Either way no block declares it, so
	// writing it is an addition; it goes to the block declaring the resource rather
	// than into a target-specific scope the user did not choose.
	if !addressesWholeElement(change) {
		return declaringBlock(blocks), nil
	}

	// The change addresses the element itself. Removing or recreating an element
	// built from several blocks cannot be expressed against just one of them.
	return sourceBlock{}, fmt.Errorf("%w: element is defined in %d blocks", errAmbiguousBlock, len(blocks))
}

// declaringBlock picks the block a resource is declared in, which is the top-level
// one whenever there is one. blocksOf and sortedBlocks both order top-level first.
func declaringBlock(blocks []sourceBlock) sourceBlock {
	for _, block := range blocks {
		if !block.override {
			return block
		}
	}
	return blocks[0]
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

	blocks := r.blocksDefiningSequence(change, step.sequencePath)
	if len(blocks) == 0 {
		return sourceBlock{}, fmt.Errorf("%w: no block defines the sequence receiving the new element", errAmbiguousBlock)
	}
	return declaringBlock(blocks), nil
}

// blocksDefiningSequence returns the blocks that write the sequence at sequencePath.
// This is for a value that does not exist yet, where only the receiving sequence is
// known; an existing value is traced through its own locations instead.
func (r *blockResolver) blocksDefiningSequence(change resolvedChange, sequencePath dyn.Path) []sourceBlock {
	var blocks []sourceBlock
	for _, block := range r.sortedBlocks() {
		blockPath, ok := r.sequencePathWithinBlock(block, change, sequencePath)
		if !ok {
			continue
		}
		sequence, err := dyn.GetByPath(r.blocks[block], r.regionPath(block, blockPath))
		if err != nil {
			continue
		}
		if slices.ContainsFunc(sequence.Locations(), func(l dyn.Location) bool { return l.File == block.file }) {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// sequencePathWithinBlock rewrites sequencePath's enclosing indices from merged
// positions to block's own, reporting false when an enclosing element is not in this
// block. This is the one place merged indices become block-local ones.
//
// change.steps is ordered outermost first, so each translated index is already known
// by the time a deeper step needs it.
func (r *blockResolver) sequencePathWithinBlock(block sourceBlock, change resolvedChange, sequencePath dyn.Path) (dyn.Path, bool) {
	result := slices.Clone(sequencePath)
	for _, step := range change.steps {
		at := len(step.sequencePath)
		if step.newElement || at >= len(sequencePath) {
			continue
		}
		// The step's own path is a prefix of sequencePath, so the indices translated
		// so far already apply to it.
		index, ok := r.indexWithinBlock(block, result[:at], step.element)
		if !ok {
			return nil, false
		}
		result[at] = dyn.Index(index)
	}
	return result, true
}

// pathWithinBlock rewrites a change's path so every sequence index addresses the
// element's position inside block rather than its position in the merged list.
func (r *blockResolver) pathWithinBlock(block sourceBlock, change resolvedChange) (*structpath.PatternNode, error) {
	nodes := change.path.AsSlice()
	indices := make([]int, len(change.steps))
	for i, step := range change.steps {
		if step.newElement {
			// Keeps the [*] placeholder; the patcher appends to the block's
			// sequence.
			indices[i] = -1
			continue
		}
		sequencePath, ok := r.sequencePathWithinBlock(block, change, step.sequencePath)
		if !ok {
			return nil, fmt.Errorf("%w: enclosing element has no position in %s", errAmbiguousBlock, block.file)
		}
		index, ok := r.indexWithinBlock(block, sequencePath, step.element)
		if !ok {
			return nil, fmt.Errorf("%w: element has no position in %s", errAmbiguousBlock, block.file)
		}
		indices[i] = index
	}

	var result *structpath.PatternNode
	next := 0
	for component, node := range nodes {
		if next < len(change.steps) && change.steps[next].component == component {
			if indices[next] < 0 {
				result = structpath.NewPatternBracketStar(result)
			} else {
				result = structpath.NewPatternIndex(result, indices[next])
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

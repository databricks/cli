package configsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

type sourceRef struct {
	file  string
	path  dyn.Path
	value dyn.Value
}

type sourceFile struct {
	content []byte
	root    dyn.Value
}

type sourceIndex struct {
	byLocation map[dyn.Location][]sourceRef
	files      map[string]*sourceFile
}

// SourceSnapshot binds physical YAML locations and file contents to the
// configuration tree used to calculate a remote-change plan.
type SourceSnapshot struct {
	index        *sourceIndex
	bundle       *bundle.Bundle
	includeFiles []string
}

// CaptureSourceSnapshot reads the source mapping before remote planning.
func CaptureSourceSnapshot(_ context.Context, b *bundle.Bundle) (*SourceSnapshot, error) {
	index, err := loadSourceIndex(b)
	if err != nil {
		return nil, err
	}
	includeFiles, err := b.MatchedIncludeFiles()
	if err != nil {
		return nil, err
	}
	return &SourceSnapshot{
		index:        index,
		bundle:       b,
		includeFiles: includeFiles,
	}, nil
}

// Validate verifies that every loaded configuration file still matches this
// snapshot, including files that do not receive a direct YAML patch.
func (s *SourceSnapshot) Validate() error {
	if s == nil {
		return errors.New("source snapshot unavailable")
	}
	currentIncludes, err := s.bundle.MatchedIncludeFiles()
	if err != nil {
		return fmt.Errorf("matching current source includes: %w", err)
	}
	if !slices.Equal(currentIncludes, s.includeFiles) {
		return fmt.Errorf("%w: source include matches changed while remote changes were being resolved", ErrSourceChanged)
	}
	for _, file := range slices.Sorted(maps.Keys(s.index.files)) {
		content, err := os.ReadFile(file)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%w: source file %s disappeared while remote changes were being resolved", ErrSourceChanged, file)
			}
			return fmt.Errorf("reading current source file %s: %w", file, err)
		}
		if !bytes.Equal(content, s.index.files[file].content) {
			return fmt.Errorf("%w: source file %s changed while remote changes were being resolved", ErrSourceChanged, file)
		}
	}
	return nil
}

func loadSourceIndex(b *bundle.Bundle) (*sourceIndex, error) {
	root := b.Config.Value()
	if !root.IsValid() {
		return nil, errors.New("source configuration unavailable")
	}

	index := &sourceIndex{
		byLocation: make(map[dyn.Location][]sourceRef),
		files:      make(map[string]*sourceFile),
	}

	fileSet := make(map[string]struct{})
	_, err := dyn.Walk(root, func(_ dyn.Path, value dyn.Value) (dyn.Value, error) {
		for _, location := range value.Locations() {
			if location.File == "" {
				continue
			}
			fileSet[location.File] = struct{}{}
		}
		return value, nil
	})
	if err != nil {
		return nil, fmt.Errorf("collecting source configuration files: %w", err)
	}
	for _, file := range slices.Sorted(maps.Keys(fileSet)) {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading source configuration file %s: %w", file, err)
		}
		parsed, diags := config.LoadFromBytes(file, content)
		if diags.HasError() {
			return nil, fmt.Errorf("parsing source configuration file %s: %w", file, diags.Error())
		}

		source := &sourceFile{content: content, root: parsed.Value()}
		index.files[file] = source
		_, err = dyn.Walk(source.root, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			for _, location := range value.Locations() {
				if location.File != file {
					continue
				}
				index.byLocation[location] = append(index.byLocation[location], sourceRef{
					file:  file,
					path:  slices.Clone(path),
					value: value,
				})
			}
			return value, nil
		})
		if err != nil {
			return nil, fmt.Errorf("indexing source configuration file %s: %w", file, err)
		}
	}

	return index, nil
}

func (s *sourceIndex) refsFor(value dyn.Value, mergedPath *structpath.PatternNode, target string) []sourceRef {
	var refs []sourceRef
	seen := make(map[string]struct{})

	for _, location := range value.Locations() {
		for _, ref := range s.byLocation[location] {
			if !sourcePathMatches(ref.path, mergedPath, target) {
				continue
			}
			key := ref.file + "\x00" + ref.path.String()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			refs = append(refs, ref)
		}
	}

	return refs
}

func sourcePathMatches(sourcePath dyn.Path, mergedPath *structpath.PatternNode, target string) bool {
	if target != "" {
		targetPrefix := dyn.NewPath(dyn.Key("targets"), dyn.Key(target))
		if relative, ok := sourcePath.CutPrefix(targetPrefix); ok {
			sourcePath = relative
		}
	}

	mergedNodes := mergedPath.AsSlice()
	if len(sourcePath) != len(mergedNodes) {
		return false
	}

	for i, node := range mergedNodes {
		if key, ok := node.StringKey(); ok {
			if sourcePath[i].Key() != key {
				return false
			}
			continue
		}
		if _, ok := node.Index(); ok {
			if sourcePath[i].Key() != "" {
				return false
			}
			continue
		}
		if node.BracketStar() {
			if sourcePath[i].Key() != "" {
				return false
			}
			continue
		}
		return false
	}

	return true
}

func sourceRefIsTarget(ref sourceRef, target string) bool {
	if target == "" {
		return false
	}
	return ref.path.HasPrefix(dyn.NewPath(dyn.Key("targets"), dyn.Key(target)))
}

func chooseSourceRef(refs []sourceRef, target string, preferTarget bool) (sourceRef, bool) {
	if len(refs) == 0 {
		return sourceRef{}, false
	}

	for _, ref := range refs {
		if sourceRefIsTarget(ref, target) == preferTarget {
			return ref, true
		}
	}
	return refs[0], true
}

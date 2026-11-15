package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// Limits of a model from code, as the server enforces them.
const (
	// maxCodePackageBytes caps the code files plus the dependency lines. The server
	// also counts the small environment file it writes, so this check rejects only
	// packages the server would reject too.
	maxCodePackageBytes = 4 * 1024 * 1024
	// maxCodeFiles caps the files besides the entry file.
	maxCodeFiles     = 1000
	maxCodePathDepth = 16
)

// pythonCacheDir holds bytecode Python writes next to imported sources.
const pythonCacheDir = "__pycache__"

// codePathSegmentRe matches a segment of a code file path the server can place.
var codePathSegmentRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`) })

// collectModelCode reads a model from code from code_source: model.entry and the
// other files a code_source snapshot would include, as UTF-8 text. Files git
// ignores aren't sent, and nor are hidden files and directories or __pycache__
// directories, whose names the server can't place or which hold no source.
// configPath locates a relative code_source.root_path.
//
// The files are stored in the model version, where anyone who can read the model
// can read them. Outside a git repository no .gitignore filters them, so the user
// must list them in code_source.include_paths.
func collectModelCode(ctx context.Context, cfg *deployConfig, configPath string) (*modelPackageCode, error) {
	root, err := resolveRootPath(ctx, cfg.CodeSource.RootPath, filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}
	isGitRepo := newGitRepo(root).isRepository(ctx)
	if !isGitRepo && len(cfg.CodeSource.IncludePaths) == 0 {
		return nil, fmt.Errorf("code_source.include_paths is required because %s isn't in a git repository: air deploy stores the files in the model version, so list the ones to send, e.g. include_paths: [%s]", filepath.ToSlash(root), *cfg.Model.Entry)
	}
	files, err := snapshotFiles(ctx, root, cfg.CodeSource.IncludePaths, isGitRepo)
	if err != nil {
		return nil, err
	}
	files = slices.DeleteFunc(files, skippedCodeFile)

	code := &modelPackageCode{EntryFile: *cfg.Model.Entry, Environment: cfg.servingEnvironment()}
	if cfg.Task != nil {
		code.Task = *cfg.Task
	}
	// Check the listing before reading anything, so a root_path that holds large
	// data fails fast instead of being read into memory.
	if err := checkCodeFiles(files, code.Environment.Dependencies); err != nil {
		return nil, err
	}

	foundEntry := false
	for _, f := range files {
		rel := filepath.ToSlash(f.rel)
		content, err := os.ReadFile(filepath.Join(root, f.rel))
		if err != nil {
			return nil, fmt.Errorf("failed to read code_source file %s: %w", rel, err)
		}
		if !utf8.Valid(content) {
			return nil, fmt.Errorf("code_source file %s isn't UTF-8 text; air deploy sends source files only, so leave it out with code_source.include_paths", rel)
		}
		if rel == code.EntryFile {
			code.EntrySource = string(content)
			foundEntry = true
			continue
		}
		code.Files = append(code.Files, modelPackageFile{Path: rel, Content: string(content)})
	}
	if !foundEntry {
		return nil, fmt.Errorf("model.entry %s isn't among the files code_source sends from %s; check that it exists, isn't ignored by git and is under code_source.include_paths", code.EntryFile, filepath.ToSlash(root))
	}
	return code, nil
}

// checkCodeFiles checks the files to send against the server's limits, from
// their listing alone.
func checkCodeFiles(files []snapshotFile, dependencies []string) error {
	// One of the files is the entry file, which the limit doesn't count.
	if len(files) > maxCodeFiles+1 {
		return fmt.Errorf("code_source sends %d files, but a model from code can include at most %d besides model.entry; narrow it with code_source.include_paths", len(files), maxCodeFiles)
	}
	var size int64
	for _, dependency := range dependencies {
		size += int64(len(dependency)) + 1
	}
	for _, f := range files {
		rel := filepath.ToSlash(f.rel)
		if !os.FileMode(f.mode).IsRegular() {
			return fmt.Errorf("code_source path %s isn't a regular file; air deploy sends the contents of files only, so replace it with a file or leave it out with code_source.include_paths", rel)
		}
		if err := validateCodePath(rel); err != nil {
			return err
		}
		size += f.size
		if size > maxCodePackageBytes {
			return fmt.Errorf("code_source files and environment.dependencies total more than %d bytes, the most a model from code can hold; narrow code_source.include_paths", maxCodePackageBytes)
		}
	}
	return nil
}

// codeFilePaths lists the files of code, entry file first, as dry runs show them.
func codeFilePaths(code *modelPackageCode) []string {
	paths := []string{code.EntryFile}
	for _, f := range code.Files {
		paths = append(paths, f.Path)
	}
	return paths
}

// sentCodePath reports whether a code_source file at rel (slash-separated) is
// sent: hidden files and directories and __pycache__ directories are skipped.
func sentCodePath(rel string) bool {
	segments := strings.Split(rel, "/")
	directories := segments[:len(segments)-1]
	return !slices.ContainsFunc(segments, isHiddenName) && !slices.Contains(directories, pythonCacheDir)
}

// skippedCodeFile reports whether a code_source file isn't sent.
func skippedCodeFile(f snapshotFile) bool {
	return !sentCodePath(filepath.ToSlash(f.rel))
}

// isHiddenName reports whether a file or directory name is hidden by convention.
func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".")
}

// validateCodePath checks that the server can place a code file at rel.
func validateCodePath(rel string) error {
	segments := strings.Split(rel, "/")
	if len(segments) <= maxCodePathDepth && !slices.ContainsFunc(segments, invalidCodePathSegment) {
		return nil
	}
	return fmt.Errorf("code_source file %s can't be sent: a path must have at most %d parts of letters, digits, '_', '-' and '.', each starting with a letter, digit or '_'; rename it or leave it out with code_source.include_paths", rel, maxCodePathDepth)
}

// invalidCodePathSegment reports whether the server can't place a path segment.
func invalidCodePathSegment(segment string) bool {
	return !codePathSegmentRe().MatchString(segment)
}

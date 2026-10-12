package localenv

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

// discoverWorkspaceRoot returns the uv workspace root when ProjectDir is a
// workspace member, and the root relative to ProjectDir (for example "../..").
// It returns "" for both when ProjectDir is not a member: a standalone project,
// the root itself, or a project uv cannot place.
//
// A folder with no pyproject.toml yet (greenfield) is not treated as a member,
// even when a members glob of a parent workspace matches it: the first run sets
// it up as a standalone project, and a re-run finds it as a member.
//
// A discovery failure (uv older than 0.9.9, uv not found under --dry-run, a
// malformed workspace) keeps the standalone behavior. A malformed workspace
// still fails later in uv sync, with uv's own message.
func (p *Pipeline) discoverWorkspaceRoot(ctx context.Context) (root, rel string) {
	if !fileExists(p.pyprojectPath()) {
		return "", ""
	}
	root, err := p.PM.WorkspaceRoot(ctx, p.ProjectDir)
	if err != nil {
		log.Debugf(ctx, CommandName+": workspace discovery failed, using the project folder: %v: %s", err, uvStderr(err))
		return "", ""
	}
	// uv prints the root with symlinks resolved and in the on-disk letter case,
	// while ProjectDir keeps the form the shell gave (on macOS, /var vs
	// /private/var, and any letter case). So compare folders by identity. Prefer
	// the root in ProjectDir's form, so the paths the command reports match the
	// folder the user ran it from. If a symlink sits between the root and
	// ProjectDir, only the symlink-resolved ProjectDir reaches the root; then use
	// uv's root, with the relative path the OS resolves.
	if ancestor, levels, ok := sameFolderAncestor(p.ProjectDir, root); ok {
		if levels == 0 {
			return "", ""
		}
		return ancestor, upLevels(levels)
	}
	resolved, err := filepath.EvalSymlinks(p.ProjectDir)
	if err != nil {
		log.Debugf(ctx, CommandName+": cannot resolve the project folder: %v", err)
		return "", ""
	}
	if _, levels, ok := sameFolderAncestor(resolved, root); ok && levels > 0 {
		return root, upLevels(levels)
	}
	log.Debugf(ctx, CommandName+": workspace root %s is not a parent of the project folder", filepath.ToSlash(root))
	return "", ""
}

// sameFolderAncestor returns the first of dir and its parents that is the same
// folder as target (by file identity, not by name), and how many levels above
// dir it is.
func sameFolderAncestor(dir, target string) (ancestor string, levels int, ok bool) {
	targetInfo, err := os.Stat(target)
	if err != nil {
		return "", 0, false
	}
	for d := dir; ; d = filepath.Dir(d) {
		if info, err := os.Stat(d); err == nil && os.SameFile(info, targetInfo) {
			return d, levels, true
		}
		if filepath.Dir(d) == d {
			return "", 0, false
		}
		levels++
	}
}

// upLevels returns the relative path n folders up, for example "../.." for 2.
func upLevels(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = ".."
	}
	return filepath.Join(parts...)
}

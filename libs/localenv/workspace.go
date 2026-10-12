package localenv

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

// discoverWorkspaceRoot returns the uv workspace root when ProjectDir is a
// workspace member, and the root relative to ProjectDir between the
// symlink-resolved folders (how the OS resolves a relative path), or "" for both
// when it is not (a standalone project, the root itself,
// or a project uv cannot place). A folder with no pyproject.toml yet (greenfield)
// is not a member: uv would walk up to a parent project, but the file this
// command creates is not listed in any workspace.
//
// A discovery failure (uv older than 0.9.9, uv not found under --dry-run, a
// malformed workspace) keeps today's behavior. A malformed workspace still fails
// later in uv sync, with uv's own message.
func (p *Pipeline) discoverWorkspaceRoot(ctx context.Context) (root, rel string) {
	if !fileExists(p.pyprojectPath()) {
		return "", ""
	}
	root, err := p.PM.WorkspaceRoot(ctx, p.ProjectDir)
	if err != nil {
		log.Debugf(ctx, CommandName+": workspace discovery failed, using the project folder: %v", err)
		return "", ""
	}
	// uv prints the symlink-resolved root, while ProjectDir comes from the shell
	// (on macOS, /var vs /private/var). Compare in resolved form.
	rel, err = resolvedRel(p.ProjectDir, root)
	if err != nil {
		log.Debugf(ctx, CommandName+": cannot compare workspace root %s with the project folder: %v", filepath.ToSlash(root), err)
		return "", ""
	}
	if rel == "." {
		return "", ""
	}
	// Prefer the root in ProjectDir's form, so the paths the command reports
	// match the folder the user ran it from. That form is only valid when no
	// symlink sits between the root and ProjectDir; otherwise use uv's root.
	joined := filepath.Join(p.ProjectDir, rel)
	if back, err := resolvedRel(joined, root); err == nil && back == "." {
		return joined, rel
	}
	return root, rel
}

// resolvedRel returns the path of target relative to base after resolving
// symlinks in both (macOS temp folders live under /var, a symlink to
// /private/var). It is "." when both name the same folder.
func resolvedRel(base, target string) (string, error) {
	rb, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	rt, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	return filepath.Rel(rb, rt)
}

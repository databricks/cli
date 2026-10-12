package localenv

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

// discoverWorkspaceRoot returns the uv workspace root when ProjectDir is a
// workspace member, or "" when it is not (a standalone project, the root itself,
// or a project uv cannot place). A folder with no pyproject.toml yet (greenfield)
// is not a member: uv would walk up to a parent project, but the file this
// command creates is not listed in any workspace.
//
// A discovery failure (uv older than 0.9.9, uv not found under --dry-run, a
// malformed workspace) keeps today's behavior. A malformed workspace still fails
// later in uv sync, with uv's own message.
func (p *Pipeline) discoverWorkspaceRoot(ctx context.Context) string {
	if !fileExists(p.pyprojectPath()) {
		return ""
	}
	root, err := p.PM.WorkspaceRoot(ctx, p.ProjectDir)
	if err != nil {
		log.Debugf(ctx, CommandName+": workspace discovery failed, using the project folder: %v", err)
		return ""
	}
	// uv prints the symlink-resolved root, while ProjectDir comes from the shell
	// (on macOS, /var vs /private/var). Compare and join in resolved form, then
	// express the root relative to ProjectDir so paths derived from it (the venv,
	// the root pyproject.toml) share ProjectDir's form.
	rel, err := resolvedRel(p.ProjectDir, root)
	if err != nil {
		log.Debugf(ctx, CommandName+": cannot compare workspace root %s with the project folder: %v", filepath.ToSlash(root), err)
		return ""
	}
	if rel == "." {
		return ""
	}
	return filepath.Join(p.ProjectDir, rel)
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

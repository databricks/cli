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
	same, err := samePath(root, p.ProjectDir)
	if err != nil {
		log.Debugf(ctx, CommandName+": cannot compare workspace root %s with the project folder: %v", filepath.ToSlash(root), err)
		return ""
	}
	if same {
		return ""
	}
	return root
}

// samePath reports whether a and b name the same folder after resolving
// symlinks (macOS temp folders live under /var, a symlink to /private/var).
func samePath(a, b string) (bool, error) {
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false, err
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		return false, err
	}
	return ra == rb, nil
}

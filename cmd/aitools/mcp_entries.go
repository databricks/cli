package aitools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
)

// agentFile is an agent config file as a command reads it: the server entries it holds, or why it cannot be read. A
// file that does not exist holds no entries.
type agentFile struct {
	entries map[string]map[string]any
	err     error
}

// agentFiles reads the agent config files of one command. An agent's current config path is found once, and each file
// is read once, so the file a record names and the file the agent reads now are each read at most once.
type agentFiles struct {
	paths map[string]agentPath
	files map[agentFileKey]agentFile
}

// agentPath is the path of an agent's config file as the agent reads it now. err is set when the path cannot be found.
type agentPath struct {
	path string
	err  error
}

// agentFileKey names a file read: the agent it is read for, and its path.
type agentFileKey struct {
	agent string
	path  string
}

func newAgentFiles() *agentFiles {
	return &agentFiles{paths: map[string]agentPath{}, files: map[agentFileKey]agentFile{}}
}

// currentPath returns the path of agent a's config file as the agent reads it now.
func (f *agentFiles) currentPath(ctx context.Context, a *agents.Agent) (string, error) {
	p, ok := f.paths[a.Name]
	if !ok {
		p.path, p.err = agentconfig.Path(ctx, a)
		f.paths[a.Name] = p
	}
	return p.path, p.err
}

// current returns agent a's config file as the agent reads it now.
func (f *agentFiles) current(ctx context.Context, a *agents.Agent) agentFile {
	path, err := f.currentPath(ctx, a)
	if err != nil {
		return agentFile{err: err}
	}
	return f.file(a, path)
}

// file returns agent a's config file at path. A record names the file an entry was written to, which need not be the
// file the agent reads now. The file is read on first use.
func (f *agentFiles) file(a *agents.Agent, path string) agentFile {
	key := agentFileKey{agent: a.Name, path: path}
	file, ok := f.files[key]
	if !ok {
		entries, err := agentconfig.ReadAt(a, path)
		file = agentFile{entries: entries, err: err}
		f.files[key] = file
	}
	return file
}

// bindingFile is the config file that a server is bound to, resolved once for the server. err is set when the file
// cannot be resolved, and then none of the server's entries can be checked for ownership.
type bindingFile struct {
	path string
	err  error
}

// resolveBinding resolves the config file that e is pinned to, or the default file when it pins none.
func resolveBinding(ctx context.Context, e mcp.Entry) bindingFile {
	path, err := mcp.ResolveConfigFile(ctx, e.ConfigFile)
	return bindingFile{path: path, err: err}
}

// checkedBinding returns the binding file of a checked server: the file the check resolved, or the error that stopped it.
func checkedBinding(check mcp.EntryCheck) bindingFile {
	if check.Path == "" {
		return bindingFile{err: check.Err}
	}
	return bindingFile{path: check.Path}
}

// owns reports whether current, the entry under e's name in an agent config, is the one this CLI wrote for e: a
// `serve` entry for its service and profile, through the binding's config file. See mcp.EntryServes.
func (b bindingFile) owns(ctx context.Context, e mcp.Entry, current map[string]any) (bool, error) {
	if b.err != nil {
		return false, b.err
	}
	return mcp.EntryServes(ctx, current, e.Service, e.Profile, b.path)
}

// changedEntryFix says how to clear an entry that was changed outside this CLI, so that remove can take the server out
// of the agent. add replaces an entry this CLI did not write only with --force. It refuses a Databricks entry that serves
// another binding whatever --force says, so that entry is deleted by hand.
func changedEntryFix(name, path string, entry map[string]any) string {
	fix := fmt.Sprintf("delete the entry from %s by hand and run `%s` again", filepath.ToSlash(path), mcp.RemoveNamesCommand(name))
	if mcpconfig.IsDatabricksEntry(entry) {
		return fix
	}
	return fix + ", or add the server again with --force"
}

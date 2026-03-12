package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

// TargetState holds deployment state for a single target.
type TargetState struct {
	Host          string
	ResourceState map[string]ResourceInfo
}

// Server is the DABs LSP server.
type Server struct {
	documents      *DocumentStore
	bundleRoot     string
	target         string
	workspaceHost  string
	resourceState  map[string]ResourceInfo
	mergedTree     dyn.Value
	allTargetState map[string]TargetState
}

// NewServer creates a new LSP server.
func NewServer() *Server {
	return &Server{
		documents:      NewDocumentStore(),
		resourceState:  make(map[string]ResourceInfo),
		allTargetState: make(map[string]TargetState),
	}
}

// Run starts the LSP server on stdin/stdout.
func (s *Server) Run(ctx context.Context) error {
	mux := handler.Map{
		"initialize":                handler.New(s.handleInitialize),
		"initialized":               handler.New(s.handleInitialized),
		"shutdown":                  handler.New(s.handleShutdown),
		"textDocument/didOpen":      handler.New(s.handleTextDocumentDidOpen),
		"textDocument/didChange":    handler.New(s.handleTextDocumentDidChange),
		"textDocument/didClose":     handler.New(s.handleTextDocumentDidClose),
		"textDocument/documentLink": handler.New(s.handleDocumentLink),
		"textDocument/hover":        handler.New(s.handleHover),
		"textDocument/definition":   handler.New(s.handleDefinition),
	}

	srv := jrpc2.NewServer(mux, &jrpc2.ServerOptions{
		AllowPush: true,
	})
	ch := channel.LSP(os.Stdin, os.Stdout)
	srv.Start(ch)
	return srv.Wait()
}

func (s *Server) handleInitialize(_ context.Context, params InitializeParams) (InitializeResult, error) {
	if params.RootURI != "" {
		s.bundleRoot = URIToPath(params.RootURI)
	} else if params.RootPath != "" {
		s.bundleRoot = params.RootPath
	}

	s.loadBundleInfo()

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full sync
			},
			HoverProvider: true,
			DocumentLinkProvider: &DocumentLinkOptions{
				ResolveProvider: false,
			},
			DefinitionProvider: true,
		},
	}, nil
}

func (s *Server) handleInitialized(_ context.Context) error {
	return nil
}

func (s *Server) handleShutdown(_ context.Context) error {
	return nil
}

func (s *Server) handleTextDocumentDidOpen(_ context.Context, params DidOpenTextDocumentParams) error {
	s.documents.Open(params.TextDocument.URI, params.TextDocument.Version, params.TextDocument.Text)
	if s.isRootConfig(params.TextDocument.URI) {
		s.loadBundleInfo()
	}
	return nil
}

func (s *Server) handleTextDocumentDidChange(_ context.Context, params DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) > 0 {
		s.documents.Change(params.TextDocument.URI, params.TextDocument.Version, params.ContentChanges[len(params.ContentChanges)-1].Text)
	}
	return nil
}

func (s *Server) handleTextDocumentDidClose(_ context.Context, params DidCloseTextDocumentParams) error {
	s.documents.Close(params.TextDocument.URI)
	return nil
}

func (s *Server) handleDocumentLink(_ context.Context, params DocumentLinkParams) ([]DocumentLink, error) {
	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	entries := IndexResources(doc)
	var links []DocumentLink
	for _, entry := range entries {
		u := s.resolveResourceURL(entry)
		if u == "" {
			continue
		}
		links = append(links, DocumentLink{
			Range:   entry.KeyRange,
			Target:  u,
			Tooltip: fmt.Sprintf("Open %s '%s' in Databricks", entry.Type, entry.Key),
		})
	}
	return links, nil
}

func (s *Server) handleHover(_ context.Context, params HoverParams) (*Hover, error) {
	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	entries := IndexResources(doc)
	for _, entry := range entries {
		if PositionInRange(params.Position, entry.KeyRange) {
			content := s.buildHoverContent(entry)
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: content,
				},
				Range: &entry.KeyRange,
			}, nil
		}
	}

	return nil, nil
}

// findRootConfig returns the path to the root bundle config file, or "" if not found.
func (s *Server) findRootConfig() string {
	for _, name := range []string{"databricks.yml", "databricks.yaml", "bundle.yml", "bundle.yaml"} {
		p := filepath.Join(s.bundleRoot, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// loadBundleInfo reads bundle config and deployment state.
func (s *Server) loadBundleInfo() {
	if s.bundleRoot == "" {
		return
	}

	configPath := s.findRootConfig()
	if configPath == "" {
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	v, err := yamlloader.LoadYAML(configPath, strings.NewReader(string(data)))
	if err != nil {
		return
	}

	if s.workspaceHost == "" {
		s.workspaceHost = LoadWorkspaceHost(v)
	}

	target := s.target
	if target == "" {
		target = LoadTarget(v)
		s.target = target
	}

	s.resourceState = LoadResourceState(s.bundleRoot, target)

	// Build URLs for resources that have IDs but no URL yet.
	if s.workspaceHost != "" {
		for key, info := range s.resourceState {
			if info.URL == "" && info.ID != "" {
				parts := strings.SplitN(key, ".", 3)
				if len(parts) == 3 {
					info.URL = BuildResourceURL(s.workspaceHost, parts[1], info.ID)
					s.resourceState[key] = info
				}
			}
		}
	}

	s.loadMergedTree(configPath, v)
	s.loadAllTargetState(v)
}

// loadMergedTree builds a merged dyn.Value from the root config and all included files.
func (s *Server) loadMergedTree(configPath string, rootValue dyn.Value) {
	s.mergedTree = rootValue

	// Extract include patterns.
	includes := rootValue.Get("include")
	if includes.Kind() != dyn.KindSequence {
		return
	}
	seq, ok := includes.AsSequence()
	if !ok {
		return
	}

	// Collect and expand glob patterns.
	seen := map[string]bool{configPath: true}
	var paths []string
	for _, item := range seq {
		pattern, ok := item.AsString()
		if !ok {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(s.bundleRoot, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				paths = append(paths, m)
			}
		}
	}
	sort.Strings(paths)

	// Parse and merge each included file.
	merged := rootValue
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := yamlloader.LoadYAML(p, strings.NewReader(string(data)))
		if err != nil {
			continue
		}
		merged, _ = merge.Merge(merged, v)
	}
	s.mergedTree = merged
}

const maxTargets = 10

// loadAllTargetState loads resource state for all targets (up to maxTargets).
func (s *Server) loadAllTargetState(v dyn.Value) {
	s.allTargetState = make(map[string]TargetState)

	targets := LoadAllTargets(v)
	if len(targets) > maxTargets {
		targets = targets[:maxTargets]
	}

	for _, t := range targets {
		host := LoadTargetWorkspaceHost(v, t)
		rs := LoadResourceState(s.bundleRoot, t)

		// Build URLs for resources with IDs.
		if host != "" {
			for key, info := range rs {
				if info.URL == "" && info.ID != "" {
					parts := strings.SplitN(key, ".", 3)
					if len(parts) == 3 {
						info.URL = BuildResourceURL(host, parts[1], info.ID)
						rs[key] = info
					}
				}
			}
		}

		s.allTargetState[t] = TargetState{
			Host:          host,
			ResourceState: rs,
		}
	}
}

func (s *Server) resolveResourceURL(entry ResourceEntry) string {
	if info, ok := s.resourceState[entry.Path]; ok {
		return info.URL
	}
	return ""
}

func (s *Server) handleDefinition(_ context.Context, params DefinitionParams) (any, error) {
	doc := s.documents.Get(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Check if cursor is on a ${...} reference.
	ref, ok := FindInterpolationAtPosition(doc.Lines, params.Position)
	if ok {
		loc, found := ResolveDefinition(s.mergedTree, ref.Path)
		if !found {
			return nil, nil
		}
		return DynLocationToLSPLocation(loc), nil
	}

	// Check if cursor is on a resource key.
	entries := IndexResources(doc)
	for _, entry := range entries {
		if PositionInRange(params.Position, entry.KeyRange) {
			refs := FindInterpolationReferences(s.mergedTree, entry.Path)
			if len(refs) == 0 {
				return nil, nil
			}
			var locs []LSPLocation
			for _, r := range refs {
				locs = append(locs, DynLocationToLSPLocation(r.Location))
			}
			return locs, nil
		}
	}

	return nil, nil
}

func (s *Server) buildHoverContent(entry ResourceEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** `%s`\n\n", entry.Type, entry.Key)

	// Show per-target state if available.
	if len(s.allTargetState) > 0 {
		hasAnyState := false
		for _, ts := range s.allTargetState {
			if _, ok := ts.ResourceState[entry.Path]; ok {
				hasAnyState = true
				break
			}
		}

		if hasAnyState {
			b.WriteString("**Targets:**\n\n")

			// Sort target names for deterministic output.
			targets := LoadAllTargets(s.mergedTree)
			for _, t := range targets {
				ts, ok := s.allTargetState[t]
				if !ok {
					continue
				}
				info, ok := ts.ResourceState[entry.Path]
				if !ok {
					continue
				}
				if info.URL != "" {
					fmt.Fprintf(&b, "- **%s**: [Open in Databricks](%s) (ID: `%s`)\n", t, info.URL, info.ID)
				} else if info.ID != "" {
					fmt.Fprintf(&b, "- **%s**: ID: `%s`\n", t, info.ID)
				}
			}
			return b.String()
		}
	}

	// Fall back to default target state.
	info, hasState := s.resourceState[entry.Path]
	if hasState && info.ID != "" {
		fmt.Fprintf(&b, "**ID:** `%s`\n\n", info.ID)
	}
	if hasState && info.Name != "" {
		fmt.Fprintf(&b, "**Name:** %s\n\n", info.Name)
	}
	if hasState && info.URL != "" {
		fmt.Fprintf(&b, "[Open in Databricks](%s)", info.URL)
	} else {
		b.WriteString("_Not yet deployed. Run `databricks bundle deploy` to create this resource._")
	}

	return b.String()
}

func (s *Server) isRootConfig(uri string) bool {
	base := filepath.Base(URIToPath(uri))
	return base == "databricks.yml" || base == "databricks.yaml" || base == "bundle.yml" || base == "bundle.yaml"
}

// SetTarget sets the target for the server.
func (s *Server) SetTarget(target string) {
	s.target = target
}

// PositionInRange checks if a position is within a range (inclusive start, exclusive end).
func PositionInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character >= r.End.Character {
		return false
	}
	return true
}

package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

// Server is the DABs LSP server.
type Server struct {
	documents     *DocumentStore
	bundleRoot    string
	target        string
	workspaceHost string
	resourceState map[string]ResourceInfo
}

// NewServer creates a new LSP server.
func NewServer() *Server {
	return &Server{
		documents:     NewDocumentStore(),
		resourceState: make(map[string]ResourceInfo),
	}
}

// Run starts the LSP server on stdin/stdout.
func (s *Server) Run(ctx context.Context) error {
	mux := handler.Map{
		"initialize":                 handler.New(s.handleInitialize),
		"initialized":               handler.New(s.handleInitialized),
		"shutdown":                   handler.New(s.handleShutdown),
		"textDocument/didOpen":       handler.New(s.handleTextDocumentDidOpen),
		"textDocument/didChange":     handler.New(s.handleTextDocumentDidChange),
		"textDocument/didClose":      handler.New(s.handleTextDocumentDidClose),
		"textDocument/documentLink":  handler.New(s.handleDocumentLink),
		"textDocument/hover":         handler.New(s.handleHover),
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

// loadBundleInfo reads bundle config and deployment state.
func (s *Server) loadBundleInfo() {
	if s.bundleRoot == "" {
		return
	}

	configPath := ""
	for _, name := range []string{"databricks.yml", "databricks.yaml", "bundle.yml", "bundle.yaml"} {
		p := filepath.Join(s.bundleRoot, name)
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}
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
}

func (s *Server) resolveResourceURL(entry ResourceEntry) string {
	if info, ok := s.resourceState[entry.Path]; ok {
		return info.URL
	}
	return ""
}

func (s *Server) buildHoverContent(entry ResourceEntry) string {
	info, hasState := s.resourceState[entry.Path]

	var b strings.Builder
	fmt.Fprintf(&b, "**%s** `%s`\n\n", entry.Type, entry.Key)

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

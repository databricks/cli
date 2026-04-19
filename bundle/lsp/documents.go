package lsp

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

// Document tracks the state of an open text document.
type Document struct {
	URI     string
	Version int
	Content string
	Lines   []string  // split by newline for position lookup
	Value   dyn.Value // parsed YAML (may be invalid)
}

// DocumentStore manages open text documents.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewDocumentStore creates an empty document store.
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]*Document)}
}

// Open registers a newly opened document.
func (s *DocumentStore) Open(uri string, version int, content string) {
	doc := &Document{
		URI:     uri,
		Version: version,
		Content: content,
		Lines:   strings.Split(content, "\n"),
	}
	doc.parse()
	s.mu.Lock()
	s.docs[uri] = doc
	s.mu.Unlock()
}

// Change updates the content of an already-open document.
func (s *DocumentStore) Change(uri string, version int, content string) {
	s.mu.Lock()
	doc, ok := s.docs[uri]
	if ok {
		doc.Version = version
		doc.Content = content
		doc.Lines = strings.Split(content, "\n")
		doc.parse()
	}
	s.mu.Unlock()
}

// Close removes a document from the store.
func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

// Get returns the document for the given URI, or nil if not found.
func (s *DocumentStore) Get(uri string) *Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.docs[uri]
}

// AllURIs returns the URIs of all open documents.
func (s *DocumentStore) AllURIs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uris := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		uris = append(uris, uri)
	}
	return uris
}

func (doc *Document) parse() {
	path := URIToPath(doc.URI)
	v, err := yamlloader.LoadYAML(path, strings.NewReader(doc.Content))
	if err != nil {
		doc.Value = dyn.InvalidValue
		return
	}
	doc.Value = v
}

// URIToPath converts a file:// URI to a filesystem path.
func URIToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	p := u.Path
	// On Windows, file URIs look like file:///C:/path
	if runtime.GOOS == "windows" && len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	return p
}

// PathToURI converts a filesystem path to a file:// URI.
func PathToURI(path string) string {
	if runtime.GOOS == "windows" {
		path = "/" + filepath.ToSlash(path)
	}
	return "file://" + path
}

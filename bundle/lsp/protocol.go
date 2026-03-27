package lsp

// InitializeParams holds the parameters sent by the client in the initialize request.
type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri"`
	RootPath  string `json:"rootPath"`
}

// InitializeResult holds the response to the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities describes the capabilities the server supports.
type ServerCapabilities struct {
	TextDocumentSync     *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	HoverProvider        bool                     `json:"hoverProvider,omitempty"`
	DocumentLinkProvider *DocumentLinkOptions     `json:"documentLinkProvider,omitempty"`
	DefinitionProvider   bool                     `json:"definitionProvider,omitempty"`
	CompletionProvider   *CompletionOptions       `json:"completionProvider,omitempty"`
}

// CompletionOptions describes options for the completion provider.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// CompletionParams holds the parameters for textDocument/completion.
type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionItem represents a single completion suggestion.
type CompletionItem struct {
	Label         string    `json:"label"`
	Kind          int       `json:"kind,omitempty"`
	Detail        string    `json:"detail,omitempty"`
	Documentation string    `json:"documentation,omitempty"`
	FilterText    string    `json:"filterText,omitempty"`
	InsertText    string    `json:"insertText,omitempty"`
	TextEdit      *TextEdit `json:"textEdit,omitempty"`
}

// TextEdit represents a text edit to be applied on completion.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// CompletionList represents a collection of completion items.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// DefinitionParams holds the parameters for textDocument/definition.
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// LSPLocation represents a location in a document (used for definition results).
type LSPLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentSyncOptions describes how text document syncing works.
type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"` // 1 = Full, 2 = Incremental
}

// DocumentLinkOptions describes options for the document link provider.
type DocumentLinkOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

// TextDocumentIdentifier identifies a text document by its URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem represents an open text document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent describes a change to a text document.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidOpenTextDocumentParams holds the parameters for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams holds the parameters for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams holds the parameters for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Position represents a zero-based line and character offset.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a span of text in a document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// DocumentLinkParams holds the parameters for textDocument/documentLink.
type DocumentLinkParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentLink represents a clickable link in a document.
type DocumentLink struct {
	Range   Range  `json:"range"`
	Target  string `json:"target"`
	Tooltip string `json:"tooltip,omitempty"`
}

// HoverParams holds the parameters for textDocument/hover.
type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Hover represents the result of a hover request.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents marked-up text for display.
type MarkupContent struct {
	Kind  string `json:"kind"` // "plaintext" or "markdown"
	Value string `json:"value"`
}

// RegistrationParams holds registrations for client/registerCapability.
type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// Registration describes a capability registration.
type Registration struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	RegisterOptions any    `json:"registerOptions,omitempty"`
}

// DidChangeWatchedFilesRegistrationOptions describes options for file watchers.
type DidChangeWatchedFilesRegistrationOptions struct {
	Watchers []FileSystemWatcher `json:"watchers"`
}

// FileSystemWatcher describes a single file system watcher.
type FileSystemWatcher struct {
	GlobPattern string `json:"globPattern"`
	Kind        int    `json:"kind,omitempty"` // 1=Create, 2=Change, 4=Delete; 7=all
}

// DidChangeWatchedFilesParams holds the notification parameters.
type DidChangeWatchedFilesParams struct {
	Changes []FileEvent `json:"changes"`
}

// FileEvent describes a file change event.
type FileEvent struct {
	URI  string `json:"uri"`
	Type int    `json:"type"` // 1=Created, 2=Changed, 3=Deleted
}

// Diagnostic severity constants.
const (
	DiagnosticSeverityWarning = 2
)

// Diagnostic represents a diagnostic (error, warning, etc.) in a document.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// PublishDiagnosticsParams holds the parameters for textDocument/publishDiagnostics.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

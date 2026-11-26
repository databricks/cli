package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// The fake stores a notebook in the format it was imported in (source or
// Jupyter), while the real /workspace/export converts it to the requested one.
//
// Recorded against the real API: the header line, COMMAND separators and
// "MAGIC %md" markdown cells of a SOURCE export, the cells and language_info of
// a JUPYTER export, and the error for a JUPYTER export of a file. Not modeled:
// cell metadata and outputs, other cell types, and HTML, DBC and R_MARKDOWN
// exports of a notebook (an explicit 500).

// notebookCommentPrefixes are the line comment markers source notebooks use.
var notebookCommentPrefixes = map[workspace.Language]string{
	workspace.LanguagePython: "#",
	workspace.LanguageR:      "#",
	workspace.LanguageScala:  "//",
	workspace.LanguageSql:    "--",
}

// commentPrefix returns the line comment marker for language, and Python's
// for a language the fake does not know.
func commentPrefix(language workspace.Language) string {
	if prefix, ok := notebookCommentPrefixes[language]; ok {
		return prefix
	}
	return notebookCommentPrefixes[workspace.LanguagePython]
}

// jupyterCell is a cell of an ipynb file. Source is a string or a list of
// strings in the format; cellSource reads both.
type jupyterCell struct {
	CellType string          `json:"cell_type"`
	Source   json.RawMessage `json:"source"`
}

func (c jupyterCell) cellSource() (string, bool) {
	var lines []string
	if err := json.Unmarshal(c.Source, &lines); err == nil {
		return strings.Join(lines, ""), true
	}
	var text string
	if err := json.Unmarshal(c.Source, &text); err == nil {
		return text, true
	}
	return "", false
}

// exportObject returns the body of entry exported in format, or a non-zero
// Response when the real API rejects the export.
func exportObject(objectPath string, entry FileEntry, format workspace.ExportFormat) ([]byte, Response) {
	if entry.Info.ObjectType != workspace.ObjectTypeNotebook {
		if format == workspace.ExportFormatJupyter {
			return nil, Response{
				StatusCode: http.StatusBadRequest,
				Body: map[string]string{
					"error_code": "BAD_REQUEST",
					"message":    fmt.Sprintf("Exporting a FILE asset at %s in JUPYTER format is not supported.", objectPath),
				},
			}
		}
		return entry.Data, Response{}
	}

	// Use the check that made the import store this body as a Jupyter notebook.
	_, isJupyter := detectJupyterNotebook(objectPath+".ipynb", entry.Data)
	if format == workspace.ExportFormatJupyter {
		if isJupyter {
			return normalizeJupyter(entry.Data, jupyterContentLanguage(entry.Data)), Response{}
		}
		return sourceToJupyter(entry.Data, entry.Info.Language), Response{}
	}
	switch format {
	case workspace.ExportFormatHtml, workspace.ExportFormatDbc, workspace.ExportFormatRMarkdown:
		return nil, Response{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("internal error: The test server does not support exporting a notebook with format=%s.", format),
		}
	case "", workspace.ExportFormatSource:
	default:
		return entry.Data, Response{}
	}
	if isJupyter {
		if source, ok := jupyterToSource(entry.Data, jupyterContentLanguage(entry.Data)); ok {
			return source, Response{}
		}
		return entry.Data, Response{}
	}
	// The real export starts a source notebook with the header line, also
	// when the notebook was imported without one.
	if !hasNotebookHeader(entry.Data) {
		prefix := commentPrefix(entry.Info.Language) + " " + notebookSourceMarker + "\n"
		return append([]byte(prefix), entry.Data...), Response{}
	}
	return entry.Data, Response{}
}

// hasNotebookHeader reports whether the first line of a source notebook is the
// header line, which is where the real import looks for it.
func hasNotebookHeader(data []byte) bool {
	header, _, _ := bytes.Cut(data, []byte("\n"))
	return bytes.Contains(header, []byte(notebookSourceMarker))
}

// marshalNotebook encodes an ipynb body without escaping HTML characters,
// which the real export leaves as they are.
func marshalNotebook(nb any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(nb); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// normalizeJupyter sets language_info.name and the Databricks metadata language
// to the notebook's content language, which the real export reports in lower
// case whatever the uploaded file said.
func normalizeJupyter(data []byte, language workspace.Language) []byte {
	var nb map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&nb); err != nil {
		return data
	}
	metadata, _ := nb["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	languageInfo, _ := metadata["language_info"].(map[string]any)
	if languageInfo == nil {
		languageInfo = map[string]any{}
	}
	languageInfo["name"] = jupyterLanguageName(language)
	metadata["language_info"] = languageInfo
	databricks, _ := metadata["application/vnd.databricks.v1+notebook"].(map[string]any)
	if databricks == nil {
		databricks = map[string]any{}
	}
	databricks["language"] = jupyterLanguageName(language)
	metadata["application/vnd.databricks.v1+notebook"] = databricks
	nb["metadata"] = metadata
	out, err := marshalNotebook(nb)
	if err != nil {
		return data
	}
	return out
}

func jupyterLanguageName(language workspace.Language) string {
	if language == "" {
		language = workspace.LanguagePython
	}
	return strings.ToLower(string(language))
}

// jupyterToSource renders an ipynb body the way the real export does for
// format=SOURCE: the header line, then the cells separated by COMMAND lines.
// Markdown cells, and code cells that start with a magic command such as %sql,
// become MAGIC comment lines.
func jupyterToSource(data []byte, language workspace.Language) ([]byte, bool) {
	var nb struct {
		Cells []jupyterCell `json:"cells"`
	}
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, false
	}
	comment := commentPrefix(language)
	cells := make([]string, 0, len(nb.Cells))
	for _, cell := range nb.Cells {
		text, ok := cell.cellSource()
		if !ok {
			return nil, false
		}
		if cell.CellType == "markdown" {
			text = "%md\n" + text
		}
		if strings.HasPrefix(text, "%") {
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				// An empty line is the bare prefix, without a trailing space.
				lines[i] = strings.TrimSuffix(comment+" MAGIC "+line, " ")
			}
			text = strings.Join(lines, "\n")
		}
		cells = append(cells, text)
	}
	separator := "\n\n" + comment + " COMMAND ----------\n\n"
	return []byte(comment + " " + notebookSourceMarker + "\n" + strings.Join(cells, separator)), true
}

// sourceCell is a cell of the ipynb body sourceToJupyter writes.
type sourceCell struct {
	CellType       string         `json:"cell_type"`
	ExecutionCount *int           `json:"execution_count,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	Outputs        *[]any         `json:"outputs,omitempty"`
	Source         []string       `json:"source"`
}

// newSourceCell builds the cell for one COMMAND-separated block, or false for
// an empty block. A block of MAGIC lines is a markdown cell for %md and a code
// cell with the magic command otherwise.
func newSourceCell(block []string, comment string) (sourceCell, bool) {
	text := strings.Trim(strings.Join(block, "\n"), "\n")
	if text == "" {
		return sourceCell{}, false
	}
	cellType := "code"
	if magic, ok := stripMagic(text, comment); ok {
		text = magic
		if rest, isMarkdown := strings.CutPrefix(text, "%md"); isMarkdown {
			cellType = "markdown"
			text = strings.TrimPrefix(rest, "\n")
		}
	}
	c := sourceCell{CellType: cellType, Metadata: map[string]any{}, Source: strings.SplitAfter(text, "\n")}
	if cellType == "code" {
		executionCount := 0
		c.ExecutionCount = &executionCount
		c.Outputs = &[]any{}
	}
	return c, true
}

// sourceToJupyter renders a source notebook the way the real export does for
// format=JUPYTER: one cell per COMMAND-separated block (empty blocks dropped),
// each split into lines that keep their trailing newline.
func sourceToJupyter(data []byte, language workspace.Language) []byte {
	comment := commentPrefix(language)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if hasNotebookHeader(data) {
		lines = lines[1:]
	}

	var cells []sourceCell
	var block []string
	for _, line := range lines {
		if strings.TrimSpace(line) == comment+" COMMAND ----------" {
			if c, ok := newSourceCell(block, comment); ok {
				cells = append(cells, c)
			}
			block = nil
			continue
		}
		block = append(block, line)
	}
	if c, ok := newSourceCell(block, comment); ok {
		cells = append(cells, c)
	}
	// A notebook without cells exports "cells": [], as the real API does.
	if cells == nil {
		cells = []sourceCell{}
	}

	nb := map[string]any{
		"cells":          cells,
		"metadata":       map[string]any{"language_info": map[string]string{"name": jupyterLanguageName(language)}},
		"nbformat":       4,
		"nbformat_minor": 0,
	}
	out, err := marshalNotebook(nb)
	if err != nil {
		panic(err)
	}
	return out
}

// stripMagic returns text without its "<comment> MAGIC " prefixes when every
// line has one.
func stripMagic(text, comment string) (string, bool) {
	prefix := comment + " MAGIC"
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == prefix {
			lines[i] = ""
			continue
		}
		rest, ok := strings.CutPrefix(line, prefix+" ")
		if !ok {
			return "", false
		}
		lines[i] = rest
	}
	return strings.Join(lines, "\n"), true
}

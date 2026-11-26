package testserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("workspace", addWorkspaceRoutes)
	registerPermissionsObjectType("notebooks", "notebook")
	registerPermissionsObjectType("directories", "directory")
	registerPermissionsObjectType("files", "file")
}

func addWorkspaceRoutes(server *Server) {
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req Request) any {
		path := req.URL.Query().Get("path")
		returnGitInfo := req.URL.Query().Get("return_git_info") == "true"
		return req.Workspace.WorkspaceGetStatus(path, returnGitInfo)
	})

	server.Handle("GET", "/api/2.0/workspace/list", func(req Request) any {
		path := req.URL.Query().Get("path")
		return req.Workspace.WorkspaceList(path)
	})

	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req Request) any {
		var request workspace.Mkdirs
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		req.Workspace.WorkspaceMkdirs(request)
		return ""
	})

	server.Handle("GET", "/api/2.0/workspace/export", func(req Request) any {
		path := req.URL.Query().Get("path")
		data := req.Workspace.WorkspaceExport(path)

		// A missing object returns 404, matching the real API; returning the nil
		// body otherwise trips the response normalizer.
		if data == nil {
			return Response{
				StatusCode: 404,
				Body:       map[string]string{"message": fmt.Sprintf("Path (%s) doesn't exist.", path)},
			}
		}

		// The filer reads the raw object body via ?direct_download=true, while
		// the SDK's Workspace.Export (used by `databricks workspace export`)
		// requests JSON and expects the base64-encoded content field.
		if req.URL.Query().Get("direct_download") == "true" {
			return data
		}

		return Response{
			Body: workspace.ExportResponse{
				Content: base64.StdEncoding.EncodeToString(data),
			},
		}
	})

	server.Handle("POST", "/api/2.0/workspace/delete", func(req Request) any {
		var request workspace.Delete
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: 500,
			}
		}
		return req.Workspace.WorkspaceDelete(request.Path, request.Recursive)
	})

	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req Request) any {
		path := req.Vars["path"]
		overwrite := req.URL.Query().Get("overwrite") == "true"
		return req.Workspace.WorkspaceFilesImportFile(path, req.Body, overwrite)
	})

	server.Handle("POST", "/api/2.0/workspace/import", func(req Request) any {
		// /workspace/import accepts both a JSON body (matching workspace.Import) and a
		// multipart form body. The multipart variant is what databricks-sdk-go's
		// Workspace.Upload uses; the JSON variant is kept for back-compat with anything
		// that still hits Workspace.Import directly.
		mediaType, _, _ := mime.ParseMediaType(req.Headers.Get("Content-Type"))
		if strings.HasPrefix(mediaType, "multipart/") {
			form, err := ParseImportForm(req)
			if err != nil {
				return Response{
					Body:       fmt.Sprintf("internal error: %s", err),
					StatusCode: http.StatusInternalServerError,
				}
			}

			// Require format explicitly: the real endpoint defaults to SOURCE, which
			// would import every upload as a notebook, so a caller that drops the
			// field is a bug rather than a request for the fake's usual behavior.
			if form.Format == "" {
				return Response{
					Body:       "internal error: The test server requires an explicit format for multipart imports.",
					StatusCode: http.StatusInternalServerError,
				}
			}
			return workspaceImport(req.Workspace, form)
		}

		var request workspace.Import
		err := json.Unmarshal(req.Body, &request)
		if err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		// The /workspace/import endpoint expects the content as base64 encoded string.
		// We need to decode it to get the actual content.
		decoded, err := base64.StdEncoding.DecodeString(request.Content)
		if err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}

		// The CLI leaves the format empty for `databricks workspace import --language X`
		// and treats empty as SOURCE (see cmd/workspace/workspace/overrides.go).
		format := string(request.Format)
		if format == "" {
			format = string(workspace.ImportFormatSource)
		}
		return workspaceImport(req.Workspace, ImportForm{
			Path:      request.Path,
			Content:   decoded,
			Format:    format,
			Language:  string(request.Language),
			Overwrite: request.Overwrite,
		})
	})

	server.Handle("GET", "/api/2.0/workspace-files/{path...}", func(req Request) any {
		path := req.Vars["path"]
		return req.Workspace.WorkspaceFilesExportFile(path)
	})
}

// Extensions that format=AUTO maps to workspace asset types /workspace/import
// cannot create, so the real endpoint rejects such uploads. Source of truth is
// workspace-hierarchy/config/AssetConfig.scala in universe.
var autoImportReservedExtensions = []string{
	".dbcluster.json",
	".dbclusterpolicy.json",
	".dbjob.json",
	".dbpipeline.json",
	".deployment.json",
	".geniedoc.json",
	".geniespace.json",
	".lakebase.json",
	".lakebridge.json",
	".lwquery.ipynb",
	".omnigentchildsession.json",
	".omnigentsession.json",
}

// autoImportUnsupported returns the response for a format=AUTO upload of filePath
// whose name the real endpoint maps to an asset type it cannot import.
func autoImportUnsupported(filePath string) (Response, bool) {
	name := strings.ToLower(filePath)
	for _, ext := range autoImportReservedExtensions {
		if strings.HasSuffix(name, ext) {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body: map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    "Requested node type [UNKNOWN] is not supported",
				},
			}, true
		}
	}
	return Response{}, false
}

// workspaceImport serves POST /api/2.0/workspace/import for both its JSON and
// multipart request bodies.
func workspaceImport(w *FakeWorkspace, form ImportForm) Response {
	var resp Response
	switch form.Format {
	case string(workspace.ImportFormatRaw):
		resp = w.WorkspaceImportRawFile(form.Path, form.Content, form.Overwrite)
	case string(workspace.ImportFormatSource):
		resp = w.WorkspaceImportNotebook(form.Path, form.Content, workspace.Language(form.Language), form.Overwrite)
	case string(workspace.ImportFormatJupyter):
		// The real endpoint reads the language from the notebook metadata.
		_, language, _ := notebook.DetectBytes(form.Path+notebook.ExtensionJupyter, form.Content)
		resp = w.WorkspaceImportNotebook(form.Path, form.Content, language, form.Overwrite)
	case string(workspace.ImportFormatAuto):
		if strings.HasSuffix(strings.ToLower(form.Path), ".zip") {
			return w.WorkspaceImportZipFolder(form.Path, form.Content, form.Overwrite)
		}
		if unsupported, ok := autoImportUnsupported(form.Path); ok {
			return unsupported
		}
		resp = w.WorkspaceImportAutoFile(form.Path, form.Content, form.Overwrite)
	default:
		return Response{
			Body:       fmt.Sprintf("internal error: The test server only supports AUTO, RAW, SOURCE and JUPYTER formats, got %q.", form.Format),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Translate any 409 from the shared fake into the 400 + errorCode
	// RESOURCE_ALREADY_EXISTS shape returned by the real /workspace/import
	// endpoint, including the AIP-193 ErrorInfo detail it attaches to path
	// collisions (universe PR #2019174, WP-6031).
	if resp.StatusCode == http.StatusConflict {
		return importAlreadyExists(fmt.Sprintf("Path (%s) already exists.", form.Path))
	}
	return resp
}

// importAlreadyExists is the /workspace/import response for a path collision.
func importAlreadyExists(message string) Response {
	return Response{
		StatusCode: http.StatusBadRequest,
		Body: map[string]any{
			"error_code": "RESOURCE_ALREADY_EXISTS",
			"message":    message,
			"details": []map[string]any{
				{
					"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
					"reason": "RESOURCE_ALREADY_EXISTS",
					"domain": "workspace.databricks.com",
				},
			},
		},
	}
}

// ImportForm holds the fields of a multipart /workspace/import request.
type ImportForm struct {
	Path      string
	Content   []byte
	Format    string
	Language  string
	Overwrite bool
}

// ParseImportForm parses the multipart body of a /workspace/import request.
func ParseImportForm(req Request) (ImportForm, error) {
	var form ImportForm
	_, params, err := mime.ParseMediaType(req.Headers.Get("Content-Type"))
	if err != nil {
		return form, err
	}
	mr := multipart.NewReader(bytes.NewReader(req.Body), params["boundary"])
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return form, nil
		}
		if err != nil {
			return form, err
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return form, err
		}
		switch part.FormName() {
		case "path":
			form.Path = string(data)
		case "content":
			form.Content = data
		case "format":
			form.Format = string(data)
		case "language":
			form.Language = string(data)
		case "overwrite":
			form.Overwrite = string(data) == "true"
		}
	}
}

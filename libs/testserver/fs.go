package testserver

import (
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("fs", addFsRoutes)
}

func addFsRoutes(server *Server) {
	server.Handle("HEAD", "/api/2.0/fs/directories/{path...}", func(req Request) any {
		dirPath := req.Vars["path"]
		if !strings.HasPrefix(dirPath, "/") {
			dirPath = "/" + dirPath
		}

		if req.Workspace.FileExists(dirPath) {
			return Response{StatusCode: 404}
		}
		if req.Workspace.DirectoryExists(dirPath) {
			return Response{StatusCode: 200}
		}
		return Response{StatusCode: 404}
	})

	server.Handle("GET", "/api/2.0/fs/directories/{path...}", func(req Request) any {
		return req.Workspace.FsListDirectory(req.Vars["path"])
	})

	server.Handle("DELETE", "/api/2.0/fs/directories/{path...}", func(req Request) any {
		return req.Workspace.FsDeleteDirectory(req.Vars["path"])
	})

	server.Handle("HEAD", "/api/2.0/fs/files/{path...}", func(req Request) any {
		path := req.Vars["path"]
		if req.Workspace.FileExists(path) {
			return Response{StatusCode: 200}
		}
		return Response{StatusCode: 404}
	})

	server.Handle("PUT", "/api/2.0/fs/directories/{path...}", func(req Request) any {
		dirPath := req.Vars["path"]
		if !strings.HasPrefix(dirPath, "/") {
			dirPath = "/" + dirPath
		}

		defer req.Workspace.LockUnlock()()

		// The API rejects creating a directory where a file already exists; it is
		// idempotent only over directories. Mirror that so callers observe the 409.
		if _, isFile := req.Workspace.files[dirPath]; isFile {
			return Response{
				StatusCode: 409,
				Body:       map[string]string{"message": "The given path points to an existing file. This API does not support operations on files."},
			}
		}

		// Create directory and all parent directories.
		for dir := dirPath; dir != "/" && dir != ""; dir = path.Dir(dir) {
			if _, exists := req.Workspace.directories[dir]; !exists {
				req.Workspace.directories[dir] = workspace.ObjectInfo{
					ObjectType: "DIRECTORY",
					Path:       dir,
					ObjectId:   nextTreenodeID(),
				}
			}
		}
		return Response{}
	})

	server.Handle("PUT", "/api/2.0/fs/files/{path...}", func(req Request) any {
		path := req.Vars["path"]
		overwrite := req.URL.Query().Get("overwrite") == "true"
		return req.Workspace.WorkspaceFilesImportFile(path, req.Body, overwrite)
	})

	server.Handle("DELETE", "/api/2.0/fs/files/{path...}", func(req Request) any {
		return req.Workspace.FsDeleteFile(req.Vars["path"])
	})

	server.Handle("GET", "/api/2.0/fs/files/{path...}", func(req Request) any {
		path := req.Vars["path"]
		data := req.Workspace.WorkspaceFilesExportFile(path)
		if data == nil {
			return Response{
				StatusCode: 404,
				Body: map[string]string{
					"message": "file does not exist",
				},
			}
		}
		return data
	})
}

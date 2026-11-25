package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("repos", addReposRoutes)
	registerStore("repos", newReposStore)
}

type reposStore struct {
	Repos        map[string]workspace.RepoInfo
	repoIdByPath map[string]int64
}

func newReposStore(*FakeWorkspace) *reposStore {
	return &reposStore{
		Repos:        map[string]workspace.RepoInfo{},
		repoIdByPath: make(map[string]int64),
	}
}

func (s *FakeWorkspace) repos() *reposStore {
	return stateOf[reposStore](s)
}

func addReposRoutes(server *Server) {
	// Repos:

	server.Handle("POST", "/api/2.0/repos", func(req Request) any {
		return req.Workspace.ReposCreate(req)
	})

	server.Handle("GET", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.repos().Repos, req.Vars["repo_id"])
	})

	server.Handle("PATCH", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return req.Workspace.ReposUpdate(req)
	})

	server.Handle("DELETE", "/api/2.0/repos/{repo_id}", func(req Request) any {
		return req.Workspace.ReposDelete(req)
	})
}

// fakeHeadCommitID is the commit the fake reports a clone is checked out to.
// Nothing is cloned here, so the remote's real HEAD is unknowable. The counting
// prefix marks it as synthetic, so it is not mistaken for a real sha.
const fakeHeadCommitID = "0a1b2c3d4e5f6a7b8e7dadd73e50a69d8ba47d8f"

func (s *FakeWorkspace) ReposCreate(req Request) Response {
	defer s.LockUnlock()()

	var repoReq workspace.CreateRepoRequest

	if err := json.Unmarshal(req.Body, &repoReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	repoId := nextTreenodeID()
	id := strconv.FormatInt(repoId, 10)

	repoInfo := workspace.RepoInfo{
		Id:           repoId,
		Path:         repoReq.Path,
		Provider:     repoReq.Provider,
		Url:          repoReq.Url,
		Branch:       "main",
		HeadCommitId: fakeHeadCommitID,
	}

	s.repos().Repos[id] = repoInfo
	s.repos().repoIdByPath[repoReq.Path] = repoId

	return Response{
		Body: repoInfo,
	}
}

func (s *FakeWorkspace) ReposUpdate(req Request) Response {
	defer s.LockUnlock()()

	var repoReq workspace.UpdateRepoRequest

	if err := json.Unmarshal(req.Body, &repoReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	repoInfo, ok := s.repos().Repos[req.Vars["repo_id"]]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	newRepoInfo := repoInfo

	if repoReq.Branch != "" {
		newRepoInfo.Branch = repoReq.Branch
	}

	s.repos().Repos[req.Vars["repo_id"]] = newRepoInfo

	return Response{
		Body: newRepoInfo,
	}
}

func (s *FakeWorkspace) ReposDelete(req Request) Response {
	defer s.LockUnlock()()

	repoInfo, ok := s.repos().Repos[req.Vars["repo_id"]]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	delete(s.repos().Repos, req.Vars["repo_id"])
	delete(s.repos().repoIdByPath, repoInfo.Path)
	return Response{}
}

// gitFolderFor returns the Git folder containing objectPath, which is the
// longest registered Git folder path that is a prefix of it. get-status reports
// the containing folder's metadata for paths inside a Git folder, not just for
// its root.
func (s *FakeWorkspace) gitFolderFor(objectPath string) (workspace.RepoInfo, bool) {
	longest := ""
	for repoPath := range s.repos().repoIdByPath {
		if objectPath != repoPath && !strings.HasPrefix(objectPath, repoPath+"/") {
			continue
		}
		if len(repoPath) > len(longest) {
			longest = repoPath
		}
	}
	if longest == "" {
		return workspace.RepoInfo{}, false
	}
	return s.repos().Repos[strconv.FormatInt(s.repos().repoIdByPath[longest], 10)], true
}

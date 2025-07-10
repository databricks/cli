package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func (s *FakeWorkspace) ReposCreate(req Request) Response {
	defer s.LockUnlock()()

	var repoReq workspace.CreateRepoRequest

	if err := json.Unmarshal(req.Body, &repoReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	s.nextRepoId++
	id := strconv.FormatInt(s.nextRepoId, 10)

	repoInfo := workspace.RepoInfo{
		Id:       s.nextRepoId,
		Path:     repoReq.Path,
		Provider: repoReq.Provider,
		Url:      repoReq.Url,
		Branch:   "main",
	}

	s.Repos[id] = repoInfo
	s.repoIdByPath[repoReq.Path] = s.nextRepoId

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

	repoInfo, ok := s.Repos[req.Vars["repo_id"]]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	newRepoInfo := repoInfo

	if repoReq.Branch != "" {
		newRepoInfo.Branch = repoReq.Branch
	}

	s.Repos[req.Vars["repo_id"]] = newRepoInfo

	return Response{
		Body: newRepoInfo,
	}
}

func (s *FakeWorkspace) ReposDelete(req Request) Response {
	defer s.LockUnlock()()

	repoInfo, ok := s.Repos[req.Vars["repo_id"]]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	delete(s.Repos, req.Vars["repo_id"])
	delete(s.repoIdByPath, repoInfo.Path)
	return Response{}
}

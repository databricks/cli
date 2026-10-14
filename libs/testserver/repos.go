package testserver

import (
	"encoding/json"
	"fmt"
	repos "github.com/databricks/cli/libs/sdktypes/repos/v1"
	"github.com/databricks/cli/libs/utils"
	"net/http"
	"strconv"
)

// fakeHeadCommitID is the commit the fake reports a clone is checked out to.
// Nothing is cloned here, so the remote's real HEAD is unknowable. The counting
// prefix marks it as synthetic, so it is not mistaken for a real sha.
const fakeHeadCommitID = "0a1b2c3d4e5f6a7b8e7dadd73e50a69d8ba47d8f"

func (s *FakeWorkspace) ReposCreate(req Request) Response {
	defer s.LockUnlock()()

	var repoReq repos.CreateRepoRequest

	if err := json.Unmarshal(req.Body, &repoReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	repoId := nextTreenodeID()
	id := strconv.FormatInt(repoId, 10)

	repoInfo := repos.RepoInfo{
		Id:           utils.NonZero(repoId),
		Path:         repoReq.Path,
		Provider:     repoReq.Provider,
		Url:          repoReq.Url,
		Branch:       utils.NonZero("main"),
		HeadCommitId: utils.NonZero(fakeHeadCommitID),
	}

	s.Repos[id] = repoInfo
	s.repoIdByPath[utils.Deref(repoReq.Path)] = repoId

	return Response{
		Body: repoInfo,
	}
}

func (s *FakeWorkspace) ReposUpdate(req Request) Response {
	defer s.LockUnlock()()

	var repoReq repos.UpdateRepoRequest

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

	if utils.Deref(repoReq.Branch) != "" {
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
	delete(s.repoIdByPath, utils.Deref(repoInfo.Path))
	return Response{}
}

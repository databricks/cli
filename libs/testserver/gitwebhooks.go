package testserver

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/gitwebhooks"
)

// Handlers for the git-webhooks API under /api/2.0/git-webhooks. Only DABs triggers are faked.

// fakeWebhookTrigger is a trigger and the webhook it is on. A repository has one webhook,
// which all its triggers share.
type fakeWebhookTrigger struct {
	webhookID string
	trigger   gitwebhooks.WebhookTrigger
}

// gitWebhooksPageSize is small so that a list of two items already takes two pages.
const gitWebhooksPageSize = 1

func gitWebhooksBadRequest(message string) Response {
	return gitWebhooksError(400, "BAD_REQUEST", message)
}

func gitWebhooksError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body:       map[string]string{"error_code": errorCode, "message": message},
	}
}

// GitWebhooksUpsert updates the trigger with the same repository, bundle path and target,
// or creates one.
func (s *FakeWorkspace) GitWebhooksUpsert(req Request) Response {
	var request gitwebhooks.UpsertWebhookTrigger
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return gitWebhooksBadRequest(fmt.Sprintf("request parsing error: %s", err))
	}
	config := request.TriggerConfig
	if message := validateGitWebhookTrigger(request.RepoURL, request.GitProvider, config); message != "" {
		return gitWebhooksBadRequest(message)
	}

	stored := *config
	if deployAs := config.DabsDeploy.DeployAs; deployAs != nil && deployAs.UserEmail != "" {
		if deployAs.UserEmail != s.MeUser(req.Token).UserName {
			return Response{StatusCode: 403, Body: map[string]string{
				"error_code": "PERMISSION_DENIED",
				"message":    "deploy_as.user_email must be your own email. To deploy as another identity, use a service principal you have the Service Principal User role on.",
			}}
		}
		// The service stores the caller, which reads show as an unset deploy_as.
		dabs := *config.DabsDeploy
		dabs.DeployAs = nil
		stored.DabsDeploy = &dabs
	}

	defer s.LockUnlock()()

	var webhookID, triggerID string
	for id, t := range s.GitWebhookTriggers {
		if t.trigger.RepoURL != request.RepoURL {
			continue
		}
		webhookID = t.webhookID
		if sameDabsTarget(t.trigger, config.DabsDeploy) {
			triggerID = id
		}
	}
	if webhookID == "" {
		webhookID = nextUUID()
	}
	isNew := triggerID == ""
	if isNew {
		triggerID = nextUUID()
	}

	stored.TriggerID = triggerID
	trigger := gitwebhooks.WebhookTrigger{
		Name:          "webhook-triggers/" + triggerID,
		RepoURL:       request.RepoURL,
		GitProvider:   request.GitProvider,
		TriggerConfig: &stored,
	}
	s.GitWebhookTriggers[triggerID] = &fakeWebhookTrigger{webhookID: webhookID, trigger: trigger}

	// A new trigger deploys the branch's head right away. The fake has no runner, so the
	// attempt stays queued.
	if isNew {
		attemptID := nextUUID()
		s.GitWebhookAttempts[triggerID] = []gitwebhooks.AutoDeployAttempt{{
			Name:          trigger.Name + "/attempts/" + attemptID,
			TriggerID:     triggerID,
			AttemptID:     attemptID,
			AttemptStatus: "ATTEMPT_STATUS_QUEUED",
			CreateTime:    nextTimestamp(),
			RepoURL:       trigger.RepoURL,
			GitProvider:   trigger.GitProvider,
			TriggerConfig: &stored,
			// The service names the runner job of every DABs attempt, from when it is queued.
			DabsResult: &gitwebhooks.DabsAttemptResult{JobID: nextID()},
		}}
	}
	return Response{Body: gitwebhooks.UpsertWebhookTriggerResponse{WebhookID: webhookID, TriggerID: triggerID}}
}

// GitWebhooksDelete deletes the trigger with the same repository, bundle path and target. Like the
// service, it succeeds when there is none.
func (s *FakeWorkspace) GitWebhooksDelete(req Request) Response {
	var request gitwebhooks.DeleteWebhookTrigger
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return gitWebhooksBadRequest(fmt.Sprintf("request parsing error: %s", err))
	}
	if message := validateGitWebhookTrigger(request.RepoURL, request.GitProvider, request.TriggerConfig); message != "" {
		return gitWebhooksBadRequest(message)
	}

	defer s.LockUnlock()()

	for id, t := range s.GitWebhookTriggers {
		if t.trigger.RepoURL == request.RepoURL && sameDabsTarget(t.trigger, request.TriggerConfig.DabsDeploy) {
			delete(s.GitWebhookTriggers, id)
		}
	}
	return Response{Body: map[string]any{}}
}

// validateGitWebhookTrigger returns what an upsert or delete request is missing, or "" if nothing.
// Both require the repo event, which delete does not match on.
func validateGitWebhookTrigger(repoURL string, provider gitwebhooks.GitProvider, config *gitwebhooks.DeploymentTriggerConfig) string {
	if repoURL == "" || provider == "" || config == nil || config.PushEvent == nil || config.DabsDeploy == nil {
		return "repo_url, git_provider, trigger_config.push_event and trigger_config.dabs_deploy are required"
	}
	if config.DabsDeploy.BundleYmlRepoPath == "" || config.DabsDeploy.DeploymentMode == "" {
		return "bundle_yml_repo_path and deployment_mode are required"
	}
	return ""
}

func sameDabsTarget(t gitwebhooks.WebhookTrigger, target *gitwebhooks.DabsDeployTarget) bool {
	stored := t.TriggerConfig.DabsDeploy
	return stored.BundleYmlRepoPath == target.BundleYmlRepoPath && stored.DeploymentMode == target.DeploymentMode
}

// GitWebhooksList lists the triggers that match the filter, ordered by name.
func (s *FakeWorkspace) GitWebhooksList(req Request) Response {
	query := req.URL.Query()
	terms, err := parseGitWebhooksFilter(query.Get("filter"))
	if err != nil {
		return gitWebhooksError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}

	defer s.LockUnlock()()

	var matched []gitwebhooks.WebhookTrigger
	for _, t := range s.GitWebhookTriggers {
		if gitWebhookTriggerMatches(t.trigger, terms) {
			matched = append(matched, t.trigger)
		}
	}
	slices.SortFunc(matched, func(a, b gitwebhooks.WebhookTrigger) int {
		return strings.Compare(a.Name, b.Name)
	})

	page, next, err := gitWebhooksPage(matched, query.Get("page_token"))
	if err != nil {
		return gitWebhooksError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	return Response{Body: gitwebhooks.ListWebhookTriggersResponse{WebhookTriggers: page, NextPageToken: next}}
}

// GitWebhooksListAttempts lists the attempts of a trigger, newest first. The fake's caller is not a
// workspace admin, so a deleted trigger's attempts are gone.
func (s *FakeWorkspace) GitWebhooksListAttempts(req Request, triggerID string) Response {
	defer s.LockUnlock()()

	if _, ok := s.GitWebhookTriggers[triggerID]; !ok {
		return gitWebhookTriggerNotFound(triggerID)
	}
	page, next, err := gitWebhooksPage(s.GitWebhookAttempts[triggerID], req.URL.Query().Get("page_token"))
	if err != nil {
		return gitWebhooksError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	return Response{Body: gitwebhooks.ListAutoDeployAttemptsResponse{AutoDeployAttempts: page, NextPageToken: next}}
}

// GitWebhooksGetAttempt returns an attempt of a trigger. Like listing, it treats the attempts of a
// deleted trigger as gone, which they are to a caller that is not a workspace admin.
func (s *FakeWorkspace) GitWebhooksGetAttempt(triggerID, attemptID string) Response {
	defer s.LockUnlock()()

	if _, ok := s.GitWebhookTriggers[triggerID]; !ok {
		return gitWebhookTriggerNotFound(triggerID)
	}
	for _, attempt := range s.GitWebhookAttempts[triggerID] {
		if attempt.AttemptID == attemptID {
			return Response{Body: attempt}
		}
	}
	return gitWebhooksError(404, "NOT_FOUND", fmt.Sprintf("Auto-deploy attempt 'webhook-triggers/%s/attempts/%s' does not exist", triggerID, attemptID))
}

func gitWebhookTriggerNotFound(triggerID string) Response {
	return gitWebhooksError(404, "NOT_FOUND", fmt.Sprintf("Webhook trigger 'webhook-triggers/%s' does not exist", triggerID))
}

// gitWebhooksPage returns the page of items that starts at the offset in token, and the token of
// the next page.
func gitWebhooksPage[T any](items []T, token string) ([]T, string, error) {
	offset := 0
	if token != "" {
		var err error
		offset, err = strconv.Atoi(token)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page_token: %s", token)
		}
	}
	offset = min(offset, len(items))
	end := min(offset+gitWebhooksPageSize, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[offset:end], next, nil
}

// parseGitWebhooksFilter parses the subset of AIP-160 that the service supports: `field = value`
// terms joined by AND, with string values in double quotes.
func parseGitWebhooksFilter(filter string) (map[string]string, error) {
	terms := map[string]string{}
	if filter == "" {
		return terms, nil
	}
	for term := range strings.SplitSeq(filter, " AND ") {
		field, value, ok := strings.Cut(term, " = ")
		if !ok {
			return nil, fmt.Errorf("unsupported filter term: %s", term)
		}
		if _, ok := gitWebhookFilterFields[field]; !ok {
			return nil, fmt.Errorf("unsupported filter field: %s", field)
		}
		// String values are quoted. git_provider is an enum, which is not.
		if field != "git_provider" {
			unquoted, err := strconv.Unquote(value)
			if err != nil || !strings.HasPrefix(value, `"`) {
				return nil, fmt.Errorf("invalid string in filter: %s", value)
			}
			value = unquoted
		}
		terms[field] = value
	}
	return terms, nil
}

func gitWebhookTriggerMatches(t gitwebhooks.WebhookTrigger, terms map[string]string) bool {
	for field, value := range terms {
		if gitWebhookFilterFields[field](t) != value {
			return false
		}
	}
	return true
}

// gitWebhookFilterFields are the filter fields that the fake supports. The fake only stores DABs
// triggers, so their DABs target is always set.
var gitWebhookFilterFields = map[string]func(gitwebhooks.WebhookTrigger) string{
	"repo_url": func(t gitwebhooks.WebhookTrigger) string {
		return t.RepoURL
	},
	"git_provider": func(t gitwebhooks.WebhookTrigger) string {
		return string(t.GitProvider)
	},
	"trigger_config.dabs_deploy.bundle_yml_repo_path": func(t gitwebhooks.WebhookTrigger) string {
		return t.TriggerConfig.DabsDeploy.BundleYmlRepoPath
	},
	"trigger_config.dabs_deploy.deployment_mode": func(t gitwebhooks.WebhookTrigger) string {
		return t.TriggerConfig.DabsDeploy.DeploymentMode
	},
}

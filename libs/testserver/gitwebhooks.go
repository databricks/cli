package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/libs/gitwebhooks"
)

// Handlers for the git-webhooks API under /api/2.0/git-webhooks. Only DABs triggers are faked.

// fakeWebhookTrigger is a trigger and the webhook it is on. A repository has one webhook,
// which all its triggers share.
type fakeWebhookTrigger struct {
	webhookID string
	trigger   gitwebhooks.WebhookTrigger
}

func gitWebhooksBadRequest(message string) Response {
	return Response{
		StatusCode: 400,
		Body:       map[string]string{"error_code": "BAD_REQUEST", "message": message},
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
	if request.RepoURL == "" || request.GitProvider == "" || config == nil || config.PushEvent == nil || config.DabsDeploy == nil {
		return gitWebhooksBadRequest("repo_url, git_provider, trigger_config.push_event and trigger_config.dabs_deploy are required")
	}
	if config.DabsDeploy.BundleYmlRepoPath == "" || config.DabsDeploy.DeploymentMode == "" {
		return gitWebhooksBadRequest("bundle_yml_repo_path and deployment_mode are required")
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
		existing := t.trigger.TriggerConfig.DabsDeploy
		if existing.BundleYmlRepoPath == config.DabsDeploy.BundleYmlRepoPath && existing.DeploymentMode == config.DabsDeploy.DeploymentMode {
			triggerID = id
		}
	}
	if webhookID == "" {
		webhookID = nextUUID()
	}
	if triggerID == "" {
		triggerID = nextUUID()
	}

	stored.TriggerID = triggerID
	s.GitWebhookTriggers[triggerID] = &fakeWebhookTrigger{
		webhookID: webhookID,
		trigger: gitwebhooks.WebhookTrigger{
			Name:          "webhook-triggers/" + triggerID,
			RepoURL:       request.RepoURL,
			GitProvider:   request.GitProvider,
			TriggerConfig: &stored,
		},
	}
	return Response{Body: gitwebhooks.UpsertWebhookTriggerResponse{WebhookID: webhookID, TriggerID: triggerID}}
}

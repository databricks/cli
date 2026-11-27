package gitwebhooks

import (
	"fmt"
	"slices"
	"strings"
)

// GitProvider is the Git provider of a repository.
type GitProvider string

const (
	GitProviderGitHub                  GitProvider = "GITHUB"
	GitProviderGitHubEnterprise        GitProvider = "GITHUB_ENTERPRISE"
	GitProviderGitLab                  GitProvider = "GITLAB"
	GitProviderGitLabEnterpriseEdition GitProvider = "GITLAB_ENTERPRISE_EDITION"
	GitProviderBitbucketCloud          GitProvider = "BITBUCKET_CLOUD"
	GitProviderBitbucketServer         GitProvider = "BITBUCKET_SERVER"
	GitProviderAzureDevOpsServices     GitProvider = "AZURE_DEV_OPS_SERVICES"
	GitProviderAwsCodeCommit           GitProvider = "AWS_CODE_COMMIT"
)

var gitProviders = []GitProvider{
	GitProviderGitHub,
	GitProviderGitHubEnterprise,
	GitProviderGitLab,
	GitProviderGitLabEnterpriseEdition,
	GitProviderBitbucketCloud,
	GitProviderBitbucketServer,
	GitProviderAzureDevOpsServices,
	GitProviderAwsCodeCommit,
}

func (f *GitProvider) String() string {
	return string(*f)
}

// Set validates v against the provider names, so GitProvider can back a flag.
func (f *GitProvider) Set(v string) error {
	if !slices.Contains(gitProviders, GitProvider(v)) {
		names := make([]string, len(gitProviders))
		for i, p := range gitProviders {
			names[i] = string(p)
		}
		return fmt.Errorf("value %q is not one of %s", v, strings.Join(names, ", "))
	}
	*f = GitProvider(v)
	return nil
}

func (f *GitProvider) Type() string {
	return "GitProvider"
}

// PushEventConfig triggers on commits pushed to a branch.
type PushEventConfig struct {
	Branch string `json:"branch,omitempty"`
}

// DeployAsIdentity is the user or service principal a deploy runs as. Exactly one field is set.
type DeployAsIdentity struct {
	UserEmail                string `json:"user_email,omitempty"`
	ServicePrincipalClientID string `json:"service_principal_client_id,omitempty"`
}

// DabsDeployTarget deploys a bundle target. The service keys a DABs trigger on
// BundleYmlRepoPath and DeploymentMode, which is the bundle target.
type DabsDeployTarget struct {
	BundleYmlRepoPath string `json:"bundle_yml_repo_path"`
	DeploymentMode    string `json:"deployment_mode"`

	// A user can only deploy as themselves, so a response sets DeployAs only to a service
	// principal. Unset means the trigger's creator.
	DeployAs *DeployAsIdentity `json:"deploy_as,omitempty"`

	// BundleDeploymentID is output only, and unset until an attempt reports it.
	BundleDeploymentID string `json:"bundle_deployment_id,omitempty"`
}

// DeploymentTriggerConfig combines the event that triggers a deploy with what it deploys.
// The service also has app and job targets, which the CLI does not set up.
type DeploymentTriggerConfig struct {
	PushEvent  *PushEventConfig  `json:"push_event,omitempty"`
	DabsDeploy *DabsDeployTarget `json:"dabs_deploy,omitempty"`

	// TriggerID is output only.
	TriggerID string `json:"trigger_id,omitempty"`
}

// WebhookTrigger is a deployment trigger on the webhook of a repository.
type WebhookTrigger struct {
	// Name has the format webhook-triggers/{trigger_id}.
	Name          string                   `json:"name,omitempty"`
	RepoURL       string                   `json:"repo_url,omitempty"`
	GitProvider   GitProvider              `json:"git_provider,omitempty"`
	TriggerConfig *DeploymentTriggerConfig `json:"trigger_config,omitempty"`
}

type UpsertWebhookTrigger struct {
	RepoURL       string                   `json:"repo_url"`
	GitProvider   GitProvider              `json:"git_provider"`
	TriggerConfig *DeploymentTriggerConfig `json:"trigger_config"`
}

type UpsertWebhookTriggerResponse struct {
	WebhookID string `json:"webhook_id,omitempty"`
	TriggerID string `json:"trigger_id,omitempty"`
}

type ListWebhookTriggersRequest struct {
	// Filter is an AIP-160 expression: `field = value` terms joined by AND.
	Filter    string `json:"-" url:"filter,omitempty"`
	PageToken string `json:"-" url:"page_token,omitempty"`
}

type ListWebhookTriggersResponse struct {
	WebhookTriggers []WebhookTrigger `json:"webhook_triggers,omitempty"`
	NextPageToken   string           `json:"next_page_token,omitempty"`
}

// DeleteWebhookTrigger deletes the trigger with the same repo_url, bundle_yml_repo_path and
// deployment_mode. The service ignores push_event, but rejects a request without it.
type DeleteWebhookTrigger struct {
	RepoURL       string                   `json:"repo_url"`
	GitProvider   GitProvider              `json:"git_provider"`
	TriggerConfig *DeploymentTriggerConfig `json:"trigger_config"`
}

// AutoDeployAttempt is one auto-deploy run of a trigger for one commit.
type AutoDeployAttempt struct {
	// Name has the format webhook-triggers/{trigger_id}/attempts/{attempt_id}.
	Name      string `json:"name,omitempty"`
	TriggerID string `json:"trigger_id,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`

	// AttemptStatus is one of ATTEMPT_STATUS_QUEUED, _RUNNING, _SUCCEEDED, _FAILED and _UNKNOWN.
	AttemptStatus string `json:"attempt_status,omitempty"`

	// ErrorMessage is set only for a failed attempt.
	ErrorMessage      string `json:"error_message,omitempty"`
	ResolvedCommitSha string `json:"resolved_commit_sha,omitempty"`

	// CreateTime and CompleteTime are RFC 3339 timestamps.
	CreateTime   string `json:"create_time,omitempty"`
	CompleteTime string `json:"complete_time,omitempty"`

	DeployAs      *DeployAsIdentity        `json:"deploy_as,omitempty"`
	RepoURL       string                   `json:"repo_url,omitempty"`
	GitProvider   GitProvider              `json:"git_provider,omitempty"`
	TriggerConfig *DeploymentTriggerConfig `json:"trigger_config,omitempty"`
	DabsResult    *DabsAttemptResult       `json:"dabs_result,omitempty"`
}

type DabsAttemptResult struct {
	// Stage is the last stage the attempt reached, such as STAGE_RUN_BUNDLE_DEPLOY. For a failed
	// attempt, it is the stage that failed.
	Stage string `json:"stage,omitempty"`

	// DeploymentVersion has the format deployments/{deployment_id}/versions/{version_id}.
	DeploymentVersion string `json:"deployment_version,omitempty"`

	// RunID and JobID are the runner job run and the runner job that the attempt ran in.
	RunID int64 `json:"run_id,omitempty"`
	JobID int64 `json:"job_id,omitempty"`
}

type ListAutoDeployAttemptsRequest struct {
	// Parent has the format webhook-triggers/{trigger_id}.
	Parent    string `json:"-" url:"-"`
	PageToken string `json:"-" url:"page_token,omitempty"`
}

type ListAutoDeployAttemptsResponse struct {
	AutoDeployAttempts []AutoDeployAttempt `json:"auto_deploy_attempts,omitempty"`
	NextPageToken      string              `json:"next_page_token,omitempty"`
}

type GetAutoDeployAttemptRequest struct {
	// Name has the format webhook-triggers/{trigger_id}/attempts/{attempt_id}.
	Name string `json:"-" url:"-"`
}

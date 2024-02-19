// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRepoSparseCheckout struct {
	Patterns []string `json:"patterns"`
}

type ResourceRepo struct {
	Branch         string                      `json:"branch,omitempty"`
	CommitHash     string                      `json:"commit_hash,omitempty"`
	GitProvider    string                      `json:"git_provider,omitempty"`
	Id             string                      `json:"id,omitempty"`
	Path           string                      `json:"path,omitempty"`
	Tag            string                      `json:"tag,omitempty"`
	Url            string                      `json:"url"`
	WorkspacePath  string                      `json:"workspace_path,omitempty"`
	SparseCheckout *ResourceRepoSparseCheckout `json:"sparse_checkout,omitempty"`
}

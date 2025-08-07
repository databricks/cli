// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCleanRoomAutoApprovalRule struct {
	AuthorCollaboratorAlias    string `json:"author_collaborator_alias,omitempty"`
	AuthorScope                string `json:"author_scope,omitempty"`
	CleanRoomName              string `json:"clean_room_name,omitempty"`
	CreatedAt                  int    `json:"created_at,omitempty"`
	RuleId                     string `json:"rule_id,omitempty"`
	RuleOwnerCollaboratorAlias string `json:"rule_owner_collaborator_alias,omitempty"`
	RunnerCollaboratorAlias    string `json:"runner_collaborator_alias,omitempty"`
}

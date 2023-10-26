// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceArtifactAllowlistArtifactMatcher struct {
	Artifact  string `json:"artifact"`
	MatchType string `json:"match_type"`
}

type ResourceArtifactAllowlist struct {
	ArtifactType    string                                     `json:"artifact_type"`
	CreatedAt       int                                        `json:"created_at,omitempty"`
	CreatedBy       string                                     `json:"created_by,omitempty"`
	Id              string                                     `json:"id,omitempty"`
	MetastoreId     string                                     `json:"metastore_id,omitempty"`
	ArtifactMatcher []ResourceArtifactAllowlistArtifactMatcher `json:"artifact_matcher,omitempty"`
}

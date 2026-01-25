// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresBranchSpec struct {
	ExpireTime       string `json:"expire_time,omitempty"`
	IsProtected      bool   `json:"is_protected,omitempty"`
	NoExpiry         bool   `json:"no_expiry,omitempty"`
	SourceBranch     string `json:"source_branch,omitempty"`
	SourceBranchLsn  string `json:"source_branch_lsn,omitempty"`
	SourceBranchTime string `json:"source_branch_time,omitempty"`
	Ttl              string `json:"ttl,omitempty"`
}

type DataSourcePostgresBranchStatus struct {
	CurrentState     string `json:"current_state,omitempty"`
	Default          bool   `json:"default,omitempty"`
	ExpireTime       string `json:"expire_time,omitempty"`
	IsProtected      bool   `json:"is_protected,omitempty"`
	LogicalSizeBytes int    `json:"logical_size_bytes,omitempty"`
	PendingState     string `json:"pending_state,omitempty"`
	SourceBranch     string `json:"source_branch,omitempty"`
	SourceBranchLsn  string `json:"source_branch_lsn,omitempty"`
	SourceBranchTime string `json:"source_branch_time,omitempty"`
	StateChangeTime  string `json:"state_change_time,omitempty"`
}

type DataSourcePostgresBranch struct {
	CreateTime string                          `json:"create_time,omitempty"`
	Name       string                          `json:"name"`
	Parent     string                          `json:"parent,omitempty"`
	Spec       *DataSourcePostgresBranchSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresBranchStatus `json:"status,omitempty"`
	Uid        string                          `json:"uid,omitempty"`
	UpdateTime string                          `json:"update_time,omitempty"`
}

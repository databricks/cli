// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresBranchesBranchesSpec struct {
	Default          bool   `json:"default,omitempty"`
	IsProtected      bool   `json:"is_protected,omitempty"`
	SourceBranch     string `json:"source_branch,omitempty"`
	SourceBranchLsn  string `json:"source_branch_lsn,omitempty"`
	SourceBranchTime string `json:"source_branch_time,omitempty"`
}

type DataSourcePostgresBranchesBranchesStatus struct {
	CurrentState     string `json:"current_state,omitempty"`
	Default          bool   `json:"default,omitempty"`
	IsProtected      bool   `json:"is_protected,omitempty"`
	LogicalSizeBytes int    `json:"logical_size_bytes,omitempty"`
	PendingState     string `json:"pending_state,omitempty"`
	SourceBranch     string `json:"source_branch,omitempty"`
	SourceBranchLsn  string `json:"source_branch_lsn,omitempty"`
	SourceBranchTime string `json:"source_branch_time,omitempty"`
	StateChangeTime  string `json:"state_change_time,omitempty"`
}

type DataSourcePostgresBranchesBranches struct {
	CreateTime string                                    `json:"create_time,omitempty"`
	Name       string                                    `json:"name"`
	Parent     string                                    `json:"parent,omitempty"`
	Spec       *DataSourcePostgresBranchesBranchesSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresBranchesBranchesStatus `json:"status,omitempty"`
	Uid        string                                    `json:"uid,omitempty"`
	UpdateTime string                                    `json:"update_time,omitempty"`
}

type DataSourcePostgresBranches struct {
	Branches []DataSourcePostgresBranchesBranches `json:"branches,omitempty"`
	PageSize int                                  `json:"page_size,omitempty"`
	Parent   string                               `json:"parent"`
}

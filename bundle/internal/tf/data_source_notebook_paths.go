// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type DataSourceNotebookPaths struct {
	Id               string `json:"id,omitempty"`
	NotebookPathList []any  `json:"notebook_path_list,omitempty"`
	Path             string `json:"path"`
	Recursive        bool   `json:"recursive"`
}

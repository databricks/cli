// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceLibrary struct {
	ClusterId    string                 `json:"cluster_id"`
	Egg          string                 `json:"egg,omitempty"`
	Id           string                 `json:"id,omitempty"`
	Jar          string                 `json:"jar,omitempty"`
	Requirements string                 `json:"requirements,omitempty"`
	Whl          string                 `json:"whl,omitempty"`
	Cran         []ResourceLibraryCran  `json:"cran,omitempty"`
	Maven        []ResourceLibraryMaven `json:"maven,omitempty"`
	Pypi         []ResourceLibraryPypi  `json:"pypi,omitempty"`
}

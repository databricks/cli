// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceClusterPolicyLibrariesCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceClusterPolicyLibrariesMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceClusterPolicyLibrariesPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceClusterPolicyLibraries struct {
	Egg          string                               `json:"egg,omitempty"`
	Jar          string                               `json:"jar,omitempty"`
	Requirements string                               `json:"requirements,omitempty"`
	Whl          string                               `json:"whl,omitempty"`
	Cran         *ResourceClusterPolicyLibrariesCran  `json:"cran,omitempty"`
	Maven        *ResourceClusterPolicyLibrariesMaven `json:"maven,omitempty"`
	Pypi         *ResourceClusterPolicyLibrariesPypi  `json:"pypi,omitempty"`
}

type ResourceClusterPolicy struct {
	Definition                      string                           `json:"definition,omitempty"`
	Description                     string                           `json:"description,omitempty"`
	Id                              string                           `json:"id,omitempty"`
	MaxClustersPerUser              int                              `json:"max_clusters_per_user,omitempty"`
	Name                            string                           `json:"name,omitempty"`
	PolicyFamilyDefinitionOverrides string                           `json:"policy_family_definition_overrides,omitempty"`
	PolicyFamilyId                  string                           `json:"policy_family_id,omitempty"`
	PolicyId                        string                           `json:"policy_id,omitempty"`
	Libraries                       []ResourceClusterPolicyLibraries `json:"libraries,omitempty"`
}

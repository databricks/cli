// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSparkVersion struct {
	Beta            bool   `json:"beta,omitempty"`
	Genomics        bool   `json:"genomics,omitempty"`
	Gpu             bool   `json:"gpu,omitempty"`
	Graviton        bool   `json:"graviton,omitempty"`
	Id              string `json:"id,omitempty"`
	Latest          bool   `json:"latest,omitempty"`
	LongTermSupport bool   `json:"long_term_support,omitempty"`
	Ml              bool   `json:"ml,omitempty"`
	Photon          bool   `json:"photon,omitempty"`
	Scala           string `json:"scala,omitempty"`
	SparkVersion    string `json:"spark_version,omitempty"`
}

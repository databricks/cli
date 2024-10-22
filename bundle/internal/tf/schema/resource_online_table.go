// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceOnlineTableSpecRunContinuously struct {
}

type ResourceOnlineTableSpecRunTriggered struct {
}

type ResourceOnlineTableSpec struct {
	PerformFullCopy     bool                                    `json:"perform_full_copy,omitempty"`
	PipelineId          string                                  `json:"pipeline_id,omitempty"`
	PrimaryKeyColumns   []string                                `json:"primary_key_columns,omitempty"`
	SourceTableFullName string                                  `json:"source_table_full_name,omitempty"`
	TimeseriesKey       string                                  `json:"timeseries_key,omitempty"`
	RunContinuously     *ResourceOnlineTableSpecRunContinuously `json:"run_continuously,omitempty"`
	RunTriggered        *ResourceOnlineTableSpecRunTriggered    `json:"run_triggered,omitempty"`
}

type ResourceOnlineTable struct {
	Id                            string                   `json:"id,omitempty"`
	Name                          string                   `json:"name"`
	Status                        []any                    `json:"status,omitempty"`
	TableServingUrl               string                   `json:"table_serving_url,omitempty"`
	UnityCatalogProvisioningState string                   `json:"unity_catalog_provisioning_state,omitempty"`
	Spec                          *ResourceOnlineTableSpec `json:"spec,omitempty"`
}

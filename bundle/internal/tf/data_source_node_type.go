// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type DataSourceNodeType struct {
	Category              string `json:"category,omitempty"`
	GbPerCore             int    `json:"gb_per_core,omitempty"`
	Graviton              bool   `json:"graviton,omitempty"`
	Id                    string `json:"id,omitempty"`
	IsIoCacheEnabled      bool   `json:"is_io_cache_enabled,omitempty"`
	LocalDisk             bool   `json:"local_disk,omitempty"`
	MinCores              int    `json:"min_cores,omitempty"`
	MinGpus               int    `json:"min_gpus,omitempty"`
	MinMemoryGb           int    `json:"min_memory_gb,omitempty"`
	PhotonDriverCapable   bool   `json:"photon_driver_capable,omitempty"`
	PhotonWorkerCapable   bool   `json:"photon_worker_capable,omitempty"`
	SupportPortForwarding bool   `json:"support_port_forwarding,omitempty"`
	Vcpu                  bool   `json:"vcpu,omitempty"`
}

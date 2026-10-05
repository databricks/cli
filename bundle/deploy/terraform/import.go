package terraform

type BindOptions struct {
	AutoApprove  bool
	ResourceType string
	ResourceKey  string
	ResourceId   string
}

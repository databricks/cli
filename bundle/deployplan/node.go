package deployplan

type ResourceNode struct {
	// Resource group in the config, e.g. "jobs" or "pipelines"
	Group string

	// Key of the resource the config
	Key string
}

// String implements StringerComparable
func (n ResourceNode) String() string {
	return n.Group + "." + n.Key
}

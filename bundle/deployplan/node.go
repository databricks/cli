package deployplan

type ResourceNode struct {
	// Resource group in the config, e.g. "jobs" or "pipelines"
	Group string

	// Key of the resource in the config, e.g. "foo" if job is located at resources.jobs.foo
	Key string
}

// String implements StringerComparable
func (n ResourceNode) String() string {
	return n.Group + "." + n.Key
}

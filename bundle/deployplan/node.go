package deployplan

type ResourceNode struct {
	Group string
	Key   string
}

// String implements StringerComparable
func (n ResourceNode) String() string {
	return n.Group + "." + n.Key
}

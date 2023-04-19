package cmdio

type NewlineEvent struct{}

func (event *NewlineEvent) String() string {
	return ""
}

func (event *NewlineEvent) IsInplaceSupported() bool {
	return false
}

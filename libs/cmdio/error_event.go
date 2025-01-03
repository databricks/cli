package cmdio

type ErrorEvent struct {
	Error string `json:"error"`
}

func (event *ErrorEvent) String() string {
	return "Error: " + event.Error
}

func (event *ErrorEvent) IsInplaceSupported() bool {
	return false
}

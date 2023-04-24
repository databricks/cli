package cmdio

import "fmt"

type ErrorEvent struct {
	Error string `json:"error"`
}

func (event *ErrorEvent) String() string {
	return fmt.Sprintf("Error: %s", event.Error)
}

func (event *ErrorEvent) IsInplaceSupported() bool {
	return false
}

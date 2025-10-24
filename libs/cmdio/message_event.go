package cmdio

type MessageEvent struct {
	Message string `json:"message"`
}

func (event *MessageEvent) String() string {
	return event.Message
}

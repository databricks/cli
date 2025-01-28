package telemetry

// RequestBody is the request body type bindings for the /telemetry-ext API endpoint.
type RequestBody struct {
	UploadTime int64    `json:"uploadTime"`
	Items      []string `json:"items"`
	ProtoLogs  []string `json:"protoLogs"`
}

// ResponseBody is the response body type bindings for the /telemetry-ext API endpoint.
type ResponseBody struct {
	Errors          []LogError `json:"errors"`
	NumProtoSuccess int64      `json:"numProtoSuccess"`
}

type LogError struct {
	Message   string `json:"message"`
	ErrorType string `json:"errorType"`
}

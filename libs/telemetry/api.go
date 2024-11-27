package telemetry

type RequestBody struct {
	UploadTime int64    `json:"uploadTime"`
	Items      []string `json:"items"`
	ProtoLogs  []string `json:"protoLogs"`
}

type ResponseBody struct {
	Errors          []LogError `json:"errors"`
	NumProtoSuccess int64      `json:"numProtoSuccess"`
}

type LogError struct {
	Message string `json:"message"`

	// Confirm with Ankit that this signature is accurate. How can I intentionally
	// trigger a error?
	ErrorType string `json:"ErrorType"`
}

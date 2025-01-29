package telemetry

// RequestBody is the request body type bindings for the /telemetry-ext API endpoint.
type RequestBody struct {
	// Timestamp in millis for when the log was uploaded.
	UploadTime int64 `json:"uploadTime"`

	// DO NOT USE. This is the legacy field for logging in usage logs (not lumberjack).
	// We keep this around because the API endpoint works only if this field is serialized
	// to an empty array.
	Items []string `json:"items"`

	// JSON encoded strings containing the proto logs. Since it's represented as a
	// string here, the values here end up being double JSON encoded in the final
	// request body.
	//
	// Any logs here will be logged in our lumberjack tables as long as a corresponding
	// protobuf is defined in universe.
	ProtoLogs []string `json:"protoLogs"`
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

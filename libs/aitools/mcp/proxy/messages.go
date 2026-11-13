package proxy

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Method names and member names the bridge reads or edits.
const (
	methodInitialize  = "initialize"
	methodInitialized = "notifications/initialized"
	methodCancelled   = "notifications/cancelled"
	methodPing        = "ping"
	methodToolsCall   = "tools/call"

	keyResult       = "result"
	keyInstructions = "instructions"
	keyCapabilities = "capabilities"
	keyTools        = "tools"
	keyListChanged  = "listChanged"
)

var (
	// errNotObject is returned for valid JSON whose top-level value is not an object.
	errNotObject = errors.New("message is not a JSON object")
	jsonNull     = json.RawMessage("null")
)

// message is the part of a JSON-RPC message the bridge reads, from either side. The rest of the message
// is forwarded as received.
type message struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
}

// errorBody is a non-2xx response body: the envelope of a JSON-RPC response, and the message of either a
// JSON-RPC error or a Databricks error body. Result is set when the body is a JSON-RPC result.
type errorBody struct {
	message
	Message string `json:"message"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

// rpcError is the error object of a JSON-RPC error response.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcErrorResponse is a JSON-RPC error response.
type rpcErrorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   rpcError        `json:"error"`
}

// parseMessage decodes the envelope of one JSON-RPC message. A syntax error is reported as a
// *json.SyntaxError; valid JSON that is not an object is errNotObject.
func parseMessage(line []byte) (message, error) {
	var m message
	if err := json.Unmarshal(line, &m); err != nil {
		return m, err
	}
	// json.Unmarshal accepts null and leaves m empty, so check the first byte.
	if !bytes.HasPrefix(bytes.TrimSpace(line), []byte{'{'}) {
		return m, errNotObject
	}
	return m, nil
}

// decodeErrorBody decodes a non-2xx body. It returns nil when the body does not decode as an object; a null body
// decodes to an empty errorBody.
func decodeErrorBody(body []byte) *errorBody {
	var e errorBody
	if json.Unmarshal(body, &e) != nil {
		return nil
	}
	return &e
}

// isRequest reports whether m is a request, which expects a response, rather than a notification.
func (m message) isRequest() bool {
	return m.Method != "" && len(m.ID) > 0 && !bytes.Equal(m.ID, jsonNull)
}

// isResponseTo reports whether m is a JSON-RPC response to request. Only a request can have one. The ids
// are compared by value (see idKey), since a server may re-encode them.
func (m message) isResponseTo(request message) bool {
	return m.Method == "" && request.isRequest() && idKey(m.ID) == idKey(request.ID)
}

// idKey returns the key that identifies a JSON-RPC id by its value: a string by its text, and a number by its exact
// value, so 7 and 7.0 match while integers above 2^53 stay distinct. Any other id is keyed by its bytes.
func idKey(raw json.RawMessage) string {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	_ = dec.Decode(&v) // an empty or invalid id has no value and keeps its bytes as the key
	switch v := v.(type) {
	case string:
		return "s" + v
	case json.Number:
		return "n" + numberKey(v.String())
	}
	return string(raw)
}

// numberKey returns the key of a JSON number by its value: its significant digits and the power of ten they are
// scaled by, so 7, 7.0 and 70e-1 share one key and -0 is 0. It works on the text, so an exponent such as 1e1000000
// costs no more than its digits. An exponent that does not fit an int64 is keyed by the text.
func numberKey(s string) string {
	s = strings.ToLower(s)
	neg := strings.HasPrefix(s, "-")
	mant, exp, _ := strings.Cut(strings.TrimPrefix(s, "-"), "e")
	whole, frac, _ := strings.Cut(mant, ".")
	scale, err := strconv.ParseInt(cmp.Or(exp, "0"), 10, 64)
	if err != nil {
		return s
	}
	scale -= int64(len(frac))
	digits := strings.TrimLeft(whole+frac, "0")
	if digits == "" {
		return "0"
	}
	trimmed := strings.TrimRight(digits, "0")
	scale += int64(len(digits) - len(trimmed))
	sign := ""
	if neg {
		sign = "-"
	}
	return sign + trimmed + "e" + strconv.FormatInt(scale, 10)
}

// initializeResultError wraps an error found while reading an initialize result.
func initializeResultError(err error) error {
	return fmt.Errorf("initialize result: %w", err)
}

// initializeProtocol returns the protocol version in the result of an initialize response. ok is false when the
// response has no result.
func initializeProtocol(data []byte) (version string, ok bool, err error) {
	var r struct {
		Result *struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", false, initializeResultError(err)
	}
	if r.Result == nil {
		return "", false, nil
	}
	return r.Result.ProtocolVersion, true, nil
}

// errorResponse builds a JSON-RPC error response. A nil id is sent as null.
func errorResponse(id json.RawMessage, code int, text string) []byte {
	if len(id) == 0 {
		id = jsonNull
	}
	// Cannot fail: the id came from a parsed message and the other fields are plain values.
	b, _ := json.Marshal(rpcErrorResponse{JSONRPC: "2.0", ID: id, Error: rpcError{Code: code, Message: text}})
	return b
}

// editInitialize appends instructions to the result's instructions and, when listChanged is set, sets
// capabilities.tools.listChanged. Other members are kept. A response without a result is returned unchanged.
func editInitialize(msg []byte, instructions string, listChanged bool) ([]byte, error) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(msg, &env); err != nil {
		return nil, err
	}
	if _, ok := env[keyResult]; !ok {
		return msg, nil
	}
	result, err := decodeObject(env[keyResult])
	if err != nil {
		return nil, initializeResultError(err)
	}
	if instructions != "" {
		var prev string
		if err := json.Unmarshal(member(result, keyInstructions), &prev); err != nil {
			return nil, fmt.Errorf("initialize instructions: %w", err)
		}
		text := instructions
		if prev != "" {
			text = prev + "\n\n" + instructions
		}
		if result[keyInstructions], err = json.Marshal(text); err != nil {
			return nil, err
		}
	}
	if listChanged {
		caps, err := decodeObject(member(result, keyCapabilities))
		if err != nil {
			return nil, fmt.Errorf("initialize capabilities: %w", err)
		}
		tools, err := decodeObject(member(caps, keyTools))
		if err != nil {
			return nil, fmt.Errorf("initialize capabilities.tools: %w", err)
		}
		tools[keyListChanged] = json.RawMessage("true")
		if caps[keyTools], err = json.Marshal(tools); err != nil {
			return nil, err
		}
		if result[keyCapabilities], err = json.Marshal(caps); err != nil {
			return nil, err
		}
	}
	if env[keyResult], err = json.Marshal(result); err != nil {
		return nil, err
	}
	return json.Marshal(env)
}

// decodeObject decodes a JSON object into its raw members. null yields an empty object.
func decodeObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	return obj, nil
}

// member returns obj[key], or null when the key is absent, so it can be decoded either way.
func member(obj map[string]json.RawMessage, key string) json.RawMessage {
	if raw, ok := obj[key]; ok {
		return raw
	}
	return jsonNull
}

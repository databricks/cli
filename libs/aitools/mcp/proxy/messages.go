package proxy

import (
	"encoding/json"
	"fmt"
)

// message is the part of a client JSON-RPC message the bridge reads. Params stay raw and are only
// decoded for the methods that need a field from them.
type message struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
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

// parseMessage decodes the envelope of one JSON-RPC message.
func parseMessage(line []byte) (message, error) {
	var m message
	err := json.Unmarshal(line, &m)
	return m, err
}

// isRequest reports whether m is a request, which expects a response, rather than a notification.
func (m message) isRequest() bool {
	return m.Method != "" && len(m.ID) > 0 && string(m.ID) != "null"
}

// errorResponse builds a JSON-RPC error response. A nil id is sent as null.
func errorResponse(id json.RawMessage, code int, text string) []byte {
	if len(id) == 0 {
		id = json.RawMessage("null")
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
	if _, ok := env["result"]; !ok {
		return msg, nil
	}
	result, err := decodeObject(env["result"])
	if err != nil {
		return nil, fmt.Errorf("initialize result: %w", err)
	}
	if instructions != "" {
		var prev string
		if err := json.Unmarshal(member(result, "instructions"), &prev); err != nil {
			return nil, fmt.Errorf("initialize instructions: %w", err)
		}
		text := instructions
		if prev != "" {
			text = prev + "\n\n" + instructions
		}
		if result["instructions"], err = json.Marshal(text); err != nil {
			return nil, err
		}
	}
	if listChanged {
		caps, err := decodeObject(member(result, "capabilities"))
		if err != nil {
			return nil, fmt.Errorf("initialize capabilities: %w", err)
		}
		tools, err := decodeObject(member(caps, "tools"))
		if err != nil {
			return nil, fmt.Errorf("initialize capabilities.tools: %w", err)
		}
		tools["listChanged"] = json.RawMessage("true")
		if caps["tools"], err = json.Marshal(tools); err != nil {
			return nil, err
		}
		if result["capabilities"], err = json.Marshal(caps); err != nil {
			return nil, err
		}
	}
	if env["result"], err = json.Marshal(result); err != nil {
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
	return json.RawMessage("null")
}

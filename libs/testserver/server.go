package testserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/databricks/cli/internal/testutil"
	"github.com/gorilla/mux"
)

type Server interface {
	Handle(method, path string, handler HandlerFunc)
	SetRequestCallback(func(request *Request))
	SetResponseCallback(func(request *Request, response *EncodedResponse))
}

type HandlerFunc func(req Request) any

type Request struct {
	Method    string
	URL       *url.URL
	Headers   http.Header
	Body      []byte
	Vars      map[string]string
	Workspace *FakeWorkspace
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       any
}

type EncodedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func NewRequest(t testutil.TestingT, r *http.Request, fakeWorkspace *FakeWorkspace) Request {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Failed to read request body: %s", err)
	}

	return Request{
		Method:    r.Method,
		URL:       r.URL,
		Headers:   r.Header,
		Body:      body,
		Vars:      mux.Vars(r),
		Workspace: fakeWorkspace,
	}
}

func normalizeResponse(t testutil.TestingT, resp any) EncodedResponse {
	result := normalizeResponseBody(t, resp)
	if result.StatusCode == 0 {
		result.StatusCode = 200
	}
	return result
}

func normalizeResponseBody(t testutil.TestingT, resp any) EncodedResponse {
	if isNil(resp) {
		t.Errorf("Handler must not return nil")
		return EncodedResponse{StatusCode: 500}
	}

	respBytes, ok := resp.([]byte)
	if ok {
		return EncodedResponse{
			Body:    respBytes,
			Headers: getHeaders(respBytes),
		}
	}

	respString, ok := resp.(string)
	if ok {
		return EncodedResponse{
			Body:    []byte(respString),
			Headers: getHeaders([]byte(respString)),
		}
	}

	respStruct, ok := resp.(Response)
	if ok {
		if isNil(respStruct.Body) {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte{},
			}
		}

		bytesVal, isBytes := respStruct.Body.([]byte)
		if isBytes {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       bytesVal,
			}
		}

		stringVal, isString := respStruct.Body.(string)
		if isString {
			return EncodedResponse{
				StatusCode: respStruct.StatusCode,
				Headers:    respStruct.Headers,
				Body:       []byte(stringVal),
			}
		}

		respBytes, err := json.MarshalIndent(respStruct.Body, "", "    ")
		if err != nil {
			t.Errorf("JSON encoding error: %s", err)
			return EncodedResponse{
				StatusCode: 500,
				Body:       []byte("internal error"),
			}
		}

		headers := respStruct.Headers
		if headers == nil {
			headers = getJsonHeaders()
		}

		return EncodedResponse{
			StatusCode: respStruct.StatusCode,
			Headers:    headers,
			Body:       respBytes,
		}
	}

	respBytes, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		t.Errorf("JSON encoding error: %s", err)
		return EncodedResponse{
			StatusCode: 500,
			Body:       []byte("internal error"),
		}
	}

	return EncodedResponse{
		Body:    respBytes,
		Headers: getJsonHeaders(),
	}
}

func getJsonHeaders() http.Header {
	return map[string][]string{
		"Content-Type": {"application/json"},
	}
}

func getHeaders(value []byte) http.Header {
	if json.Valid(value) {
		return getJsonHeaders()
	} else {
		return map[string][]string{
			"Content-Type": {"text/plain"},
		}
	}
}

func getToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	prefix := "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return header[len(prefix):]
}

func isNil(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

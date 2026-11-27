package testserver

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Request fault kinds, each a backend failure we have seen from a real workspace. Status kinds
// answer instead of the handler, so the request has no effect. Kinds ending in "-after" run the
// handler first and lose its response, as when a gateway times out or a connection drops after
// the backend applied the request.
const (
	FaultTooManyRequests    = "429"
	FaultInternalError      = "500"
	FaultBadGateway         = "502"
	FaultUnavailable        = "503"
	FaultGatewayTimeout     = "504"
	FaultGatewayTimeoutLate = "504-after"
	FaultReset              = "reset"
	FaultResetLate          = "reset-after"
	FaultTruncatedLate      = "eof-after"
)

// RequestFaultKinds lists every kind SetRequestFault accepts.
var RequestFaultKinds = []string{
	FaultTooManyRequests,
	FaultInternalError,
	FaultBadGateway,
	FaultUnavailable,
	FaultGatewayTimeout,
	FaultGatewayTimeoutLate,
	FaultReset,
	FaultResetLate,
	FaultTruncatedLate,
}

// faultStatusResponses are the responses of the status kinds. The bodies match what the workspace
// API (JSON) and its Envoy gateway (plain text) send.
var faultStatusResponses = map[string]EncodedResponse{
	FaultTooManyRequests: jsonFaultResponse(http.StatusTooManyRequests, "REQUEST_LIMIT_EXCEEDED", "Injected fault: too many requests."),
	FaultInternalError:   jsonFaultResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "Injected fault: internal error."),
	FaultBadGateway:      textFaultResponse(http.StatusBadGateway, "upstream connect error or disconnect/reset before headers. reset reason: connection termination"),
	FaultUnavailable:     jsonFaultResponse(http.StatusServiceUnavailable, "TEMPORARILY_UNAVAILABLE", "Injected fault: service unavailable."),
	FaultGatewayTimeout:  textFaultResponse(http.StatusGatewayTimeout, "upstream request timeout"),
}

func jsonFaultResponse(status int, errorCode, message string) EncodedResponse {
	return EncodedResponse{
		StatusCode: status,
		Headers:    getJsonHeaders(),
		Body:       fmt.Appendf(nil, `{"error_code": %q, "message": %q}`, errorCode, message),
	}
}

func textFaultResponse(status int, body string) EncodedResponse {
	return EncodedResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": {"text/plain"}},
		Body:       []byte(body),
	}
}

// FiredRequestFault describes the request a fault was injected into.
type FiredRequestFault struct {
	At     int    `json:"at"`
	Kind   string `json:"kind"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

// requestFault fails the at-th request the server receives (1-based) with the given kind. With at
// set to 0 it only counts requests.
type requestFault struct {
	mu    sync.Mutex
	count int
	at    int
	kind  string
	fired *FiredRequestFault
}

// next counts a request and returns the fault kind to inject into it, or "" for none.
func (f *requestFault) next(method, path string) string {
	// Scripts set up kill and fault rules through these; they are not CLI traffic.
	if strings.HasPrefix(path, "/__testserver/") {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	if f.at == 0 || f.count != f.at {
		return ""
	}
	f.fired = &FiredRequestFault{At: f.at, Kind: f.kind, Method: method, Path: path}
	return f.kind
}

// SetRequestFault makes the server fail the at-th request it receives (1-based) with kind, one
// of RequestFaultKinds. Requests to /__testserver/ endpoints are not counted.
func (s *Server) SetRequestFault(at int, kind string) error {
	if at < 1 {
		return fmt.Errorf("request number must be at least 1, got %d", at)
	}
	if !slices.Contains(RequestFaultKinds, kind) {
		return fmt.Errorf("unknown request fault kind %q, expected one of %s", kind, strings.Join(RequestFaultKinds, ", "))
	}
	s.requestFault.mu.Lock()
	defer s.requestFault.mu.Unlock()
	s.requestFault.at = at
	s.requestFault.kind = kind
	return nil
}

// RequestCount returns the number of requests the server received, not counting requests to
// /__testserver/ endpoints.
func (s *Server) RequestCount() int {
	s.requestFault.mu.Lock()
	defer s.requestFault.mu.Unlock()
	return s.requestFault.count
}

// FiredRequestFault returns the request the fault set with SetRequestFault was injected into, or
// nil if the server has not received that many requests.
func (s *Server) FiredRequestFault() *FiredRequestFault {
	s.requestFault.mu.Lock()
	defer s.requestFault.mu.Unlock()
	return s.requestFault.fired
}

// faultRunsHandler reports whether the handler applies the request before the fault is injected.
func faultRunsHandler(kind string) bool {
	return strings.HasSuffix(kind, "-after")
}

// writeRequestFault sends the fault of the given kind instead of resp, the handler's response for
// the kinds that run it.
func (s *Server) writeRequestFault(w http.ResponseWriter, request *Request, kind string, resp EncodedResponse) {
	switch kind {
	case FaultGatewayTimeoutLate:
		s.writeResponse(w, request, faultStatusResponses[FaultGatewayTimeout])
	case FaultReset, FaultResetLate:
		s.notifyResponse(request, EncodedResponse{})
		conn := s.hijack(w)
		if conn == nil {
			return
		}
		// A zero linger makes Close send RST instead of FIN, so the client reads
		// "connection reset by peer".
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.SetLinger(0)
		}
		_ = conn.Close()
	case FaultTruncatedLate:
		s.notifyResponse(request, EncodedResponse{})
		conn := s.hijack(w)
		if conn == nil {
			return
		}
		// Promise more bytes than are sent, so the client reads "unexpected EOF" in the body.
		sent := resp.Body[:len(resp.Body)/2]
		var head strings.Builder
		fmt.Fprintf(&head, "HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		for key, values := range resp.Headers {
			for _, value := range values {
				fmt.Fprintf(&head, "%s: %s\r\n", key, value)
			}
		}
		head.WriteString("Content-Length: " + strconv.Itoa(len(resp.Body)+1) + "\r\n\r\n")
		_, _ = conn.Write(append([]byte(head.String()), sent...))
		_ = conn.Close()
	default:
		s.writeResponse(w, request, faultStatusResponses[kind])
	}
}

func (s *Server) hijack(w http.ResponseWriter) net.Conn {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		s.t.Errorf("request fault: response writer does not support hijacking")
		return nil
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		s.t.Errorf("request fault: hijack failed: %s", err)
		return nil
	}
	return conn
}

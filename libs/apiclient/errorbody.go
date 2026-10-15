package apiclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	stringErrorRegex = regexp.MustCompile(`^([A-Z_]+): (.*)$`)
	htmlMessageRegex = regexp.MustCompile(`<pre>(.*)</pre>`)
)

// normalizeErrorBody rewrites error responses that are not JSON objects into
// the standard {"error_code", "message"} shape, so sdk-go reports the same
// message and error code as the previous SDK did: proxies and some legacy
// endpoints answer with an empty body, plain text or an HTML page.
func normalizeErrorBody(resp *http.Response) error {
	if resp.StatusCode < 400 || resp.Body == nil {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	if normalized, ok := errorBodyFromText(resp, body); ok {
		body = normalized
		resp.Header.Set("Content-Type", "application/json")
		resp.Header.Del("Content-Length")
		resp.ContentLength = int64(len(body))
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

func errorBodyFromText(resp *http.Response, body []byte) ([]byte, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(body, &obj) == nil {
		return nil, false
	}
	var errorCode, message string
	trimmed := bytes.TrimSpace(body)
	switch {
	case len(trimmed) == 0:
		message = http.StatusText(resp.StatusCode)
	case stringErrorRegex.Match(trimmed):
		m := stringErrorRegex.FindSubmatch(trimmed)
		errorCode, message = string(m[1]), string(m[2])
	case htmlMessageRegex.Match(body):
		message = strings.Trim(string(htmlMessageRegex.FindSubmatch(body)[1]), " .")
		errorCode = "UNKNOWN"
		if _, status, ok := strings.Cut(resp.Status, " "); ok && strings.TrimSpace(status) != "" {
			errorCode = strings.ReplaceAll(strings.ToUpper(strings.Trim(status, " .")), " ", "_")
		}
	default:
		errorCode, message = "UNKNOWN", string(body)
	}
	out := map[string]string{"message": message}
	if errorCode != "" {
		out["error_code"] = errorCode
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, false
	}
	return data, true
}

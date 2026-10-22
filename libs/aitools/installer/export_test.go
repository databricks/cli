package installer

import "net/http"

// SetHTTPClientForTest lets command regression tests exercise skills resolution
// and downloads against a local server. It returns a function restoring it.
func SetHTTPClientForTest(client *http.Client) func() {
	orig := httpClient
	httpClient = func() *http.Client { return client }
	return func() { httpClient = orig }
}

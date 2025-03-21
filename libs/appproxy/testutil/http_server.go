package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewHttpServer creates a new httptest.Server that will respond with the same body that was sent to it
// and will check if the headers match the expected headers.
func NewHttpServer(t *testing.T, expectHeaders map[string]string) *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		// Read all headers and check if they match the expected headers
		for k, v := range expectHeaders {
			if v != r.Header.Get(k) {
				http.Error(w, "Internal Server Error, headers don't match", http.StatusInternalServerError)
				return
			}
		}

		// read the JSON request body and write it back
		_, err := io.Copy(w, r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}))

	return server
}

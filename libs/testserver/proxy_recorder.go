package testserver

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ProxyRecorder represents a proxy server that records HTTP traffic
type ProxyRecorder struct {
	// Remote server to proxy to
	RemoteURL string
	// File to record traffic to
	RecordFile *os.File
	// Lock for concurrent writes to the file
	mu sync.Mutex
	// The test server
	Server *httptest.Server
}

// NewProxyRecorder creates a new proxy recorder
func NewProxyRecorder(remoteURL, recordFilePath string) (*ProxyRecorder, error) {
	// Open or create the record file
	file, err := os.OpenFile(recordFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open record file: %w", err)
	}

	// Create the proxy recorder
	pr := &ProxyRecorder{
		RemoteURL:  remoteURL,
		RecordFile: file,
	}

	// Create the test server
	pr.Server = httptest.NewServer(http.HandlerFunc(pr.proxyHandler))

	return pr, nil
}

// Close stops the server and closes the record file
func (pr *ProxyRecorder) Close() {
	pr.Server.Close()
	err := pr.RecordFile.Close()
	if err != nil {
		log.Fatalf("Error closing record file")
	}
}

// URL returns the URL of the test server
func (pr *ProxyRecorder) URL() string {
	return pr.Server.URL
}

// proxyHandler handles HTTP requests, forwards them to the remote server,
// and records both the request and response
func (pr *ProxyRecorder) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Record timestamp
	timestamp := time.Now().Format(time.RFC3339)

	// Dump the request to be recorded
	var requestDump strings.Builder
	requestDump.WriteString(fmt.Sprintf("=== REQUEST %s ===\n", timestamp))
	requestDump.WriteString(fmt.Sprintf("%s %s %s\n", r.Method, r.URL.Path, r.Proto))

	// Record headers
	for name, values := range r.Header {
		for _, value := range values {
			requestDump.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}

	// Read and record the request body, if any
	var requestBody []byte
	if r.Body != nil {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		// Close the original body and replace it with a new reader
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(requestBody))

		if len(requestBody) > 0 {
			requestDump.WriteString("\n")

			// Check content type to handle binary data appropriately
			contentType := r.Header.Get("Content-Type")
			if isBinaryContent(contentType) {
				// For binary content, log length and format instead of raw content
				requestDump.WriteString(fmt.Sprintf("[Binary data, %d bytes, Content-Type: %s]\n",
					len(requestBody), contentType))
			} else {
				// For text content, convert to UTF-8 if needed
				bodyStr := safeStringConversion(requestBody)
				requestDump.WriteString(bodyStr)
			}
		}
	}
	requestDump.WriteString("\n\n")

	// Create a new request to the remote server
	remoteURL := pr.RemoteURL + r.URL.Path
	if r.URL.RawQuery != "" {
		remoteURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, remoteURL, bytes.NewBuffer(requestBody))
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the request to the remote server
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to proxy request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Record the response
	var responseDump strings.Builder
	responseDump.WriteString(fmt.Sprintf("=== RESPONSE %s ===\n", timestamp))
	responseDump.WriteString(fmt.Sprintf("%s %d %s\n", resp.Proto, resp.StatusCode, resp.Status))

	// Record response headers
	for name, values := range resp.Header {
		for _, value := range values {
			responseDump.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}

	// Read and record the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	var bodyForLogging []byte
	isGzipped := false

	// Check if response is gzipped
	for _, encoding := range resp.Header.Values("Content-Encoding") {
		if strings.Contains(strings.ToLower(encoding), "gzip") {
			isGzipped = true
			break
		}
	}

	if isGzipped {
		// Decompress for logging
		gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			// If decompression fails, just log that it's compressed
			responseDump.WriteString(fmt.Sprintf("\n[Gzipped content, %d bytes]\n", len(responseBody)))
		} else {
			defer gzipReader.Close()

			// Read the decompressed content
			decompressed, err := io.ReadAll(gzipReader)
			if err != nil {
				responseDump.WriteString(fmt.Sprintf("\n[Error decompressing gzipped content: %v]\n", err))
			} else {
				bodyForLogging = decompressed
				responseDump.WriteString("\n[Decompressed from gzip]\n")
			}
		}
	} else {
		bodyForLogging = responseBody
	}

	if len(bodyForLogging) > 0 {
		// Check content type to handle binary data appropriately
		contentType := resp.Header.Get("Content-Type")
		if isBinaryContent(contentType) {
			// For binary content, log length and format instead of raw content
			responseDump.WriteString(fmt.Sprintf("[Binary data, %d bytes, Content-Type: %s]\n",
				len(responseBody), contentType))
		} else {
			// For text content, convert to UTF-8 if needed
			bodyStr := safeStringConversion(bodyForLogging)
			responseDump.WriteString(bodyStr)
		}
	}
	responseDump.WriteString("\n\n")

	// Write the request and response to the record file
	pr.mu.Lock()
	pr.RecordFile.WriteString(requestDump.String())
	pr.RecordFile.WriteString(responseDump.String())
	pr.mu.Unlock()

	// Copy the response headers to the original response writer
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code
	w.WriteHeader(resp.StatusCode)

	// Write the response body to the original response writer
	w.Write(responseBody)
}

// ensureValidUTF8 converts a string to valid UTF-8, replacing invalid byte sequences
func ensureValidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Replace invalid UTF-8 sequences with the Unicode replacement character (U+FFFD)
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				// Invalid UTF-8 sequence, replace with Unicode replacement character
				v = append(v, '\uFFFD')
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// isBinaryContent checks if the content type represents binary data
func isBinaryContent(contentType string) bool {
	if contentType == "" {
		return false
	}

	// List of common binary content types
	binaryTypes := []string{
		"image/", "audio/", "video/", "application/octet-stream",
		"application/pdf", "application/zip", "application/gzip",
		"application/x-tar", "application/x-rar-compressed",
		"application/x-7z-compressed", "application/x-msdownload",
		"application/vnd.ms-", "application/x-ms",
	}

	for _, prefix := range binaryTypes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}

	return false
}

// safeStringConversion attempts to convert bytes to a valid UTF-8 string
func safeStringConversion(data []byte) string {
	// First try direct conversion
	s := string(data)

	// Check if it's valid UTF-8
	if utf8.ValidString(s) {
		return s
	}

	// Try to detect encoding and convert to UTF-8
	// For simplicity, we'll just replace invalid characters
	return ensureValidUTF8(s)
}

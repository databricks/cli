package files

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newControlEngine builds an engine whose control plane points at an httptest
// server running h.
func newControlEngine(t *testing.T, h http.HandlerFunc) *engine {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return newEngine(buildUploadClient(t, srv.URL, srv.Client(), ""))
}

func TestInitiate(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/2.0/fs/files/") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("action"); got != "initiate-upload" {
			t.Errorf("action = %q", got)
		}
		if got := r.URL.Query().Get("overwrite"); got != "true" {
			t.Errorf("overwrite = %q, want true", got)
		}
		_, _ = io.WriteString(w, `{"multipart_upload":{"session_token":"tok"}}`)
	})

	ow := true
	res, err := c.initiateUpload(t.Context(), "/Volumes/c/s/v/f.bin", &ow)
	if err != nil {
		t.Fatal(err)
	}
	if res.MultipartUpload == nil || res.MultipartUpload.SessionToken != "tok" {
		t.Errorf("multipart = %+v", res.MultipartUpload)
	}
	if res.ResumableUpload != nil {
		t.Error("resumable should be nil")
	}
}

func TestCreatePartURLs(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"upload_part_urls":[{"url":"https://cloud.test/part/1","part_number":1,"headers":[{"name":"x-h","value":"v"}]}]}`)
	})

	urls, err := c.createPartURLs(t.Context(), "/Volumes/c/s/v/f.bin", "tok", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0].URL != "https://cloud.test/part/1" || urls[0].PartNumber != 1 || urls[0].Headers["x-h"] != "v" {
		t.Errorf("url = %+v", urls[0])
	}
}

func TestCreatePartURLsEmpty(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"upload_part_urls":[]}`)
	})
	if _, err := c.createPartURLs(t.Context(), "/Volumes/c/s/v/f.bin", "tok", 1, 1); !errors.Is(err, errUnexpectedServerResponse) {
		t.Fatalf("err = %v, want errUnexpectedServerResponse", err)
	}
}

func TestCreateResumableURL(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"resumable_upload_url":{"url":"https://cloud.test/resumable","headers":[{"name":"x-h","value":"v"}]}}`)
	})

	purl, err := c.createResumableURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if purl.URL != "https://cloud.test/resumable" || purl.Headers["x-h"] != "v" {
		t.Errorf("url = %+v", purl)
	}
}

func TestCreateResumableURLMissing(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})
	if _, err := c.createResumableURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok"); !errors.Is(err, errUnexpectedServerResponse) {
		t.Fatalf("err = %v, want errUnexpectedServerResponse", err)
	}
}

func TestCreateAbortURL(t *testing.T) {
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"abort_upload_url":{"url":"https://cloud.test/abort"}}`)
	})

	purl, err := c.createAbortURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if purl.URL != "https://cloud.test/abort" {
		t.Errorf("url = %q", purl.URL)
	}
}

func TestCompleteMultipartSortsParts(t *testing.T) {
	var got []int
	c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
		if action := r.URL.Query().Get("action"); action != "complete-upload" {
			t.Errorf("action = %q", action)
		}
		var body struct {
			Parts []struct {
				PartNumber int    `json:"part_number"`
				ETag       string `json:"etag"`
			} `json:"parts"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, p := range body.Parts {
			got = append(got, p.PartNumber)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.completeMultipart(t.Context(), "/Volumes/c/s/v/f.bin", "tok", map[int]string{3: "e3", 1: "e1", 2: "e2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("parts = %v, want sorted [1 2 3]", got)
	}
}

func TestControlPlaneAlreadyExists(t *testing.T) {
	conflict := func(errorCode string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"error_code":"`+errorCode+`","message":"the file being created already exists"}`)
		}
	}

	// complete-upload (the multipart finish) returning 409 ALREADY_EXISTS maps to
	// the sentinel so callers can skip-if-exists, matching single-shot/resumable.
	c := newControlEngine(t, conflict("ALREADY_EXISTS"))
	if err := c.completeMultipart(t.Context(), "/Volumes/c/s/v/f.bin", "tok", map[int]string{1: "e1"}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("completeMultipart err = %v, want ErrAlreadyExists", err)
	}

	// initiate-upload maps the same way.
	c2 := newControlEngine(t, conflict("ALREADY_EXISTS"))
	if _, err := c2.initiateUpload(t.Context(), "/Volumes/c/s/v/f.bin", nil); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("initiateUpload err = %v, want ErrAlreadyExists", err)
	}

	// A different 409 is not mistaken for the already-exists sentinel.
	c3 := newControlEngine(t, conflict("RESOURCE_CONFLICT"))
	if _, err := c3.initiateUpload(t.Context(), "/Volumes/c/s/v/f.bin", nil); errors.Is(err, ErrAlreadyExists) {
		t.Errorf("a non-ALREADY_EXISTS 409 mapped to ErrAlreadyExists: %v", err)
	}
}

func TestSingleShotUploadOverwrite(t *testing.T) {
	// The single-shot octet-stream PUT streams the raw body and sends the
	// overwrite query only when explicitly set.
	yes, no := true, false
	cases := []struct {
		name      string
		overwrite *bool
		wantQuery string
	}{
		{"unset", nil, ""},
		{"true", &yes, "true"},
		{"false", &no, "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotBody, gotOverwrite, gotContentType string
			c := newControlEngine(t, func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				gotBody = string(b)
				gotOverwrite = r.URL.Query().Get("overwrite")
				gotContentType = r.Header.Get("Content-Type")
				w.WriteHeader(http.StatusOK)
			})
			err := c.uploadSingleShot(t.Context(), "/Volumes/c/s/v/f.bin", tc.overwrite, strings.NewReader("hello"))
			if err != nil {
				t.Fatal(err)
			}
			if gotBody != "hello" {
				t.Errorf("body = %q, want the raw octet stream %q", gotBody, "hello")
			}
			if gotContentType != "application/octet-stream" {
				t.Errorf("Content-Type = %q, want application/octet-stream", gotContentType)
			}
			if gotOverwrite != tc.wantQuery {
				t.Errorf("overwrite query = %q, want %q", gotOverwrite, tc.wantQuery)
			}
		})
	}
}

package files

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config/credentials"
	sdkfiles "github.com/databricks/databricks-sdk-go/service/files"
)

func noAuth() credentials.CredentialsProvider {
	return credentials.CredentialsProviderFn(func(r *http.Request) error { return nil })
}

// fakeFiles is a fake SDK Files client that records the single-shot request and
// returns a canned error.
type fakeFiles struct {
	req sdkfiles.UploadRequest
	err error
}

func (f *fakeFiles) Upload(ctx context.Context, req sdkfiles.UploadRequest) error {
	f.req = req
	return f.err
}

// newClient builds a Client whose control plane points at an httptest server
// running h, and returns the fake SDK Files client backing single-shot.
func newClient(t *testing.T, h http.HandlerFunc) (*Client, *fakeFiles) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	ff := &fakeFiles{}
	c, err := New(ClientOptions{FilesClient: ff, Host: srv.URL, CredentialsProvider: noAuth()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, ff
}

func TestNewValidation(t *testing.T) {
	if _, err := New(ClientOptions{Host: "https://h.test", CredentialsProvider: noAuth()}); err == nil {
		t.Error("nil SDK client should error")
	}
	if _, err := New(ClientOptions{FilesClient: &fakeFiles{}, CredentialsProvider: noAuth()}); err == nil {
		t.Error("empty host should error")
	}
	if _, err := New(ClientOptions{FilesClient: &fakeFiles{}, Host: "https://h.test"}); err == nil {
		t.Error("nil creds should error")
	}
}

func TestInitiate(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	res, err := c.Initiate(t.Context(), "/Volumes/c/s/v/f.bin", &ow)
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
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"upload_part_urls":[{"url":"https://cloud.test/part/1","part_number":1,"headers":[{"name":"x-h","value":"v"}]}]}`)
	})

	urls, err := c.CreatePartURLs(t.Context(), "/Volumes/c/s/v/f.bin", "tok", 1, 1)
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
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"upload_part_urls":[]}`)
	})
	if _, err := c.CreatePartURLs(t.Context(), "/Volumes/c/s/v/f.bin", "tok", 1, 1); !errors.Is(err, ErrUnexpectedServerResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedServerResponse", err)
	}
}

func TestCreateResumableURL(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"resumable_upload_url":{"url":"https://cloud.test/resumable","headers":[{"name":"x-h","value":"v"}]}}`)
	})

	purl, err := c.CreateResumableURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if purl.URL != "https://cloud.test/resumable" || purl.Headers["x-h"] != "v" {
		t.Errorf("url = %+v", purl)
	}
}

func TestCreateResumableURLMissing(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})
	if _, err := c.CreateResumableURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok"); !errors.Is(err, ErrUnexpectedServerResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedServerResponse", err)
	}
}

func TestCreateAbortURL(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"abort_upload_url":{"url":"https://cloud.test/abort"}}`)
	})

	purl, err := c.CreateAbortURL(t.Context(), "/Volumes/c/s/v/f.bin", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if purl.URL != "https://cloud.test/abort" {
		t.Errorf("url = %q", purl.URL)
	}
}

func TestCompleteMultipartSortsParts(t *testing.T) {
	var got []int
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("action"); got != "complete-upload" {
			t.Errorf("action = %q", got)
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

	err := c.CompleteMultipart(t.Context(), "/Volumes/c/s/v/f.bin", "tok", map[int]string{3: "e3", 1: "e1", 2: "e2"})
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
	c, _ := newClient(t, conflict("ALREADY_EXISTS"))
	if err := c.CompleteMultipart(t.Context(), "/Volumes/c/s/v/f.bin", "tok", map[int]string{1: "e1"}); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("CompleteMultipart err = %v, want ErrAlreadyExists", err)
	}

	// initiate-upload maps the same way.
	c2, _ := newClient(t, conflict("ALREADY_EXISTS"))
	if _, err := c2.Initiate(t.Context(), "/Volumes/c/s/v/f.bin", nil); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("Initiate err = %v, want ErrAlreadyExists", err)
	}

	// A different 409 is not mistaken for the already-exists sentinel.
	c3, _ := newClient(t, conflict("RESOURCE_CONFLICT"))
	if _, err := c3.Initiate(t.Context(), "/Volumes/c/s/v/f.bin", nil); errors.Is(err, ErrAlreadyExists) {
		t.Errorf("a non-ALREADY_EXISTS 409 mapped to ErrAlreadyExists: %v", err)
	}
}

func TestUploadBuildsRequest(t *testing.T) {
	c, ff := newClient(t, func(w http.ResponseWriter, r *http.Request) {})
	ow := false
	err := c.Upload(t.Context(), "/Volumes/c/s/v/f.bin", &ow, strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if ff.req.FilePath != "/Volumes/c/s/v/f.bin" {
		t.Errorf("FilePath = %q", ff.req.FilePath)
	}
	// overwrite=false is sent explicitly via ForceSendFields.
	if ff.req.Overwrite != false || len(ff.req.ForceSendFields) != 1 || ff.req.ForceSendFields[0] != "Overwrite" {
		t.Errorf("overwrite not forced: %+v", ff.req.ForceSendFields)
	}
}

func TestUploadOverwriteUnset(t *testing.T) {
	c, ff := newClient(t, func(w http.ResponseWriter, r *http.Request) {})
	if err := c.Upload(t.Context(), "/Volumes/c/s/v/f.bin", nil, strings.NewReader("hi")); err != nil {
		t.Fatal(err)
	}
	if len(ff.req.ForceSendFields) != 0 {
		t.Errorf("ForceSendFields = %v, want empty when overwrite is unset", ff.req.ForceSendFields)
	}
}

func TestUploadAlreadyExists(t *testing.T) {
	c, ff := newClient(t, func(w http.ResponseWriter, r *http.Request) {})
	ff.err = &apierr.APIError{StatusCode: http.StatusConflict, ErrorCode: "ALREADY_EXISTS", Message: "exists"}

	err := c.Upload(t.Context(), "/Volumes/c/s/v/f.bin", nil, strings.NewReader("hi"))
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestUploadOtherErrorPassThrough(t *testing.T) {
	c, ff := newClient(t, func(w http.ResponseWriter, r *http.Request) {})
	ff.err = &apierr.APIError{StatusCode: http.StatusForbidden, ErrorCode: "PERMISSION_DENIED"}

	err := c.Upload(t.Context(), "/Volumes/c/s/v/f.bin", nil, strings.NewReader("hi"))
	if errors.Is(err, ErrAlreadyExists) {
		t.Fatal("a 403 must not map to ErrAlreadyExists")
	}
	if err == nil {
		t.Fatal("expected the underlying error to pass through")
	}
}

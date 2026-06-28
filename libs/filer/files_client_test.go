package filer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"testing"

	"github.com/databricks/cli/libs/env"
	uploadfiles "github.com/databricks/cli/libs/upload/files"
)

// onlyReader hides the Seek method of an underlying reader, modelling a
// non-seekable stream (e.g. a remote download body).
type onlyReader struct{ io.Reader }

func TestIsSeekable(t *testing.T) {
	in := []byte("hello, files API")

	if !isSeekable(bytes.NewReader(in)) {
		t.Fatal("bytes.Reader should be seekable")
	}

	// The position must be left at the start so a subsequent read covers every byte.
	r := bytes.NewReader(in)
	if !isSeekable(r) {
		t.Fatal("bytes.Reader should be seekable")
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, in) {
		t.Errorf("read %q after isSeekable, want %q (position not left at start)", b, in)
	}

	if isSeekable(onlyReader{bytes.NewReader(in)}) {
		t.Error("a non-seekable reader should report false")
	}
}

func TestMultipartUploadEnabled(t *testing.T) {
	ctx := t.Context()
	if MultipartUploadEnabled(ctx) {
		t.Error("multipart upload must be disabled by default")
	}
	for _, on := range []string{"true", "1", "yes", "on"} {
		if !MultipartUploadEnabled(env.Set(ctx, multipartUploadEnvVar, on)) {
			t.Errorf("value %q should enable multipart upload", on)
		}
	}
	for _, off := range []string{"false", "0", "", "nonsense"} {
		if MultipartUploadEnabled(env.Set(ctx, multipartUploadEnvVar, off)) {
			t.Errorf("value %q should not enable multipart upload", off)
		}
	}
}

func TestMapUploadError(t *testing.T) {
	const p = "/Volumes/c/s/v/f.bin"

	if err := mapUploadError(nil, p); err != nil {
		t.Errorf("nil error should pass through, got %v", err)
	}

	// The engine's already-exists sentinel (even wrapped) must surface as fs.ErrExist
	// so skip-if-exists keeps working.
	for _, in := range []error{
		uploadfiles.ErrAlreadyExists,
		fmt.Errorf("upload failed: %w", uploadfiles.ErrAlreadyExists),
	} {
		got := mapUploadError(in, p)
		if !errors.Is(got, fs.ErrExist) {
			t.Errorf("mapUploadError(%v) = %v, want errors.Is fs.ErrExist", in, got)
		}
	}

	// Other errors pass through unchanged.
	other := errors.New("boom")
	if got := mapUploadError(other, p); got != other {
		t.Errorf("mapUploadError(other) = %v, want it unchanged", got)
	}
}

func TestWithUploadConcurrency(t *testing.T) {
	var cfg filesClientConfig
	WithUploadConcurrency(64)(&cfg)
	if cfg.uploadConcurrency != 64 {
		t.Errorf("uploadConcurrency = %d, want 64", cfg.uploadConcurrency)
	}
}

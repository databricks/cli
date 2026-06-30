package cloudstorage

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBytesBody(t *testing.T) {
	data := []byte("hello world")
	b := BytesBody(data)

	if b.Size() != int64(len(data)) {
		t.Errorf("Size = %d, want %d", b.Size(), len(data))
	}

	// Reader is called once per attempt and must re-supply the full body from the
	// start each time, so a retry needs no rewind.
	for i := range 2 {
		r, err := b.Reader()
		if err != nil {
			t.Fatalf("attempt %d: Reader returned error: %v", i, err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("attempt %d: ReadAll returned error: %v", i, err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("attempt %d: read %q, want %q", i, got, data)
		}
	}
}

func TestSectionBody(t *testing.T) {
	src := strings.NewReader("0123456789")
	b := SectionBody(src, 3, 4)

	if b.Size() != 4 {
		t.Errorf("Size = %d, want 4", b.Size())
	}

	// Each Reader reads only the [offset, offset+length) section, from the start.
	for i := range 2 {
		r, err := b.Reader()
		if err != nil {
			t.Fatalf("attempt %d: Reader returned error: %v", i, err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("attempt %d: ReadAll returned error: %v", i, err)
		}
		if string(got) != "3456" {
			t.Errorf("attempt %d: read %q, want %q", i, got, "3456")
		}
	}
}

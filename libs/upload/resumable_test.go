package upload

import "testing"

func TestExtractRangeOffset(t *testing.T) {
	n, ok, err := extractRangeOffset("bytes=0-99")
	noErr(t, err)
	if !ok || n != 99 {
		t.Errorf("extractRangeOffset(bytes=0-99) = (%d, %v), want (99, true)", n, ok)
	}

	_, ok, err = extractRangeOffset("")
	noErr(t, err)
	if ok {
		t.Error("empty header should report nothing confirmed")
	}

	if _, _, err := extractRangeOffset("garbage"); err == nil {
		t.Error("malformed header should error")
	}
}

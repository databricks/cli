package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// magic marks the inflated payload as a bundle telemetry bitmap.
const magic = "DBTB"

// ContextFullBundle marks a bitmap covering the whole user-authored config. Other
// context values are reserved for narrower bitmaps in the future.
const ContextFullBundle uint16 = 0

// headerSize is the size of the fixed header: magic (4) + size (4) + context (2).
const headerSize = 10

// encodedPrefix starts every encoded bitmap: "bm1.<base64>". The logging
// pipeline's data shape (DATA_SHAPE_DABS_BUNDLE_BITMAP) matches the prefix and
// bounds the length, so the bitmap survives centralization from regulated
// (FedRAMP) workspaces, where strings without a shape are redacted. The "1" is
// the format version.
const encodedPrefix = "bm1."

// MaxSize is the largest bitmap Encode takes, in bits. Its encoding fits in
// MaxEncodedBodyLen.
const MaxSize = 32768

// MaxEncodedBodyLen is the longest base64 body the shape accepts after "bm1.":
// DEFLATE's worst case for a MaxSize payload plus a margin, base64-encoded.
// Encode never exceeds it (TestEncodeFitsShape).
const MaxEncodedBodyLen = 5504

// Encode packs bits into the wire format (magic, size, context, bitmap),
// compresses it with DEFLATE, base64-encodes the result, and prefixes it with
// "bm1.". DEFLATE is used rather than gzip because it has no timestamp, keeping
// the output deterministic.
func Encode(bits []bool, context uint16) (string, error) {
	if len(bits) == 0 || len(bits) > MaxSize {
		return "", fmt.Errorf("bitmap has %d bits; the encoding takes 1 to %d", len(bits), MaxSize)
	}
	n := len(bits)
	raw := make([]byte, headerSize+(n+7)/8)
	copy(raw, magic)
	binary.BigEndian.PutUint32(raw[4:], uint32(n))
	binary.BigEndian.PutUint16(raw[8:], context)
	for i, set := range bits {
		if set {
			raw[headerSize+i/8] |= 1 << uint(7-i%8)
		}
	}

	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(raw); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return encodedPrefix + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode reverses Encode: it checks the "bm1." prefix, base64-decodes,
// inflates, validates the header, and returns the bits and context.
func Decode(encoded string) (bits []bool, context uint16, err error) {
	body, ok := strings.CutPrefix(encoded, encodedPrefix)
	if !ok {
		return nil, 0, fmt.Errorf("missing %q prefix", encodedPrefix)
	}
	compressed, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, 0, fmt.Errorf("base64: %w", err)
	}
	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, 0, fmt.Errorf("inflate: %w", err)
	}
	if len(raw) < headerSize {
		return nil, 0, fmt.Errorf("payload too short: %d bytes", len(raw))
	}
	if string(raw[:4]) != magic {
		return nil, 0, fmt.Errorf("bad magic %q", raw[:4])
	}
	n := binary.BigEndian.Uint32(raw[4:])
	context = binary.BigEndian.Uint16(raw[8:])
	// Compare in uint64 so a corrupt size near 2^32 cannot overflow int on 32-bit platforms.
	if uint64(len(raw)-headerSize) != (uint64(n)+7)/8 {
		return nil, 0, fmt.Errorf("payload size mismatch: %d bits, %d bytes", n, len(raw))
	}
	bits = make([]bool, n)
	for i := range bits {
		bits[i] = raw[headerSize+i/8]&(1<<uint(7-i%8)) != 0
	}
	return bits, context, nil
}

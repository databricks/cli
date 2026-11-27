package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Sizes around byte boundaries exercise the MSB-first packing and padding.
	for _, n := range []int{1, 7, 8, 9, 16, 17, MaxSize} {
		bits := make([]bool, n)
		for i := range bits {
			bits[i] = i%3 == 0
		}
		for _, context := range []uint16{ContextFullBundle, 0xBEEF} {
			encoded, err := Encode(bits, context)
			require.NoError(t, err)

			decoded, gotContext, err := Decode(encoded)
			require.NoError(t, err)
			assert.Equal(t, context, gotContext)
			assert.Len(t, decoded, n)
			assert.Equal(t, bits, decoded)
		}
	}
	// The shape that lets the bitmap leave regulated regions takes 1 to MaxSize bits.
	for _, n := range []int{0, MaxSize + 1} {
		_, err := Encode(make([]bool, n), ContextFullBundle)
		assert.ErrorContains(t, err, "the encoding takes 1 to 32768", n)
	}
}

func TestEncodeBitOrder(t *testing.T) {
	encoded, err := Encode([]bool{true, false, false, false, false, false, false, true, true}, ContextFullBundle)
	require.NoError(t, err)

	body, ok := strings.CutPrefix(encoded, "bm1.")
	require.True(t, ok, encoded)
	compressed, err := base64.StdEncoding.DecodeString(body)
	require.NoError(t, err)
	var raw bytes.Buffer
	_, err = raw.ReadFrom(flate.NewReader(bytes.NewReader(compressed)))
	require.NoError(t, err)

	assert.Equal(t, []byte{'D', 'B', 'T', 'B', 0, 0, 0, 9, 0, 0, 0b1000_0001, 0b1000_0000}, raw.Bytes())
}

// deflateBase64 wraps a raw payload the way Encode does, so Decode's header
// checks can be exercised on hand-built payloads.
func deflateBase64(t *testing.T, raw []byte) string {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	require.NoError(t, err)
	_, err = w.Write(raw)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func header(magic string, size uint32) []byte {
	raw := make([]byte, headerSize)
	copy(raw, magic)
	binary.BigEndian.PutUint32(raw[4:], size)
	return raw
}

func TestDecodeErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		encoded string
		err     string
	}{
		{
			name:    "missing prefix",
			encoded: deflateBase64(t, append(header(magic, 1), 0x80)),
			err:     `missing "bm1." prefix`,
		},
		{
			name:    "bad base64",
			encoded: "bm1.not-valid-base64!!!",
			err:     "base64",
		},
		{
			name:    "not deflate",
			encoded: "bm1." + base64.StdEncoding.EncodeToString([]byte{0xff, 0xff}),
			err:     "inflate",
		},
		{
			name:    "short header",
			encoded: "bm1." + deflateBase64(t, []byte("DBTB")),
			err:     "payload too short",
		},
		{
			name:    "bad magic",
			encoded: "bm1." + deflateBase64(t, append(header("XXXX", 1), 0x80)),
			err:     "bad magic",
		},
		{
			name:    "missing bitmap bytes",
			encoded: "bm1." + deflateBase64(t, header(magic, 9)),
			err:     "payload size mismatch",
		},
		{
			name:    "extra bitmap bytes",
			encoded: "bm1." + deflateBase64(t, append(header(magic, 1), 0x80, 0x00)),
			err:     "payload size mismatch",
		},
		{
			name:    "size near 2^32",
			encoded: "bm1." + deflateBase64(t, header(magic, 0xFFFFFFFF)),
			err:     "payload size mismatch",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Decode(tc.encoded)
			assert.ErrorContains(t, err, tc.err)
		})
	}
}

// validShape is a copy of DATA_SHAPE_DABS_BUNDLE_BITMAP from universe
// compliance/shape: the format the logging pipeline lets out of regulated
// (FedRAMP) regions. A bitmap that fails it is redacted there.
func validShape(v string) bool {
	body, ok := strings.CutPrefix(v, "bm1.")
	if !ok || len(body) < 4 || len(body)%4 != 0 || len(body) > MaxEncodedBodyLen {
		return false
	}
	for i, c := range body {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '+', c == '/':
		case c == '=' && i >= len(body)-2 && (i == len(body)-1 || body[len(body)-1] == '='):
		default:
			return false
		}
	}
	return true
}

func TestEncodeFitsShape(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	sizes := []int{1, 2, 7, 8, 9, 100, len(EmbeddedSchema()), MaxSize}
	for range 50 {
		sizes = append(sizes, 1+r.IntN(MaxSize))
	}
	for _, size := range sizes {
		// Random bits compress worst; all set and none set are the extremes.
		for _, density := range []float64{0, 0.05, 0.5, 1} {
			bits := make([]bool, size)
			for i := range bits {
				bits[i] = r.Float64() < density
			}
			encoded, err := Encode(bits, ContextFullBundle)
			require.NoError(t, err)
			assert.True(t, validShape(encoded), "size %d density %v: %q fails the shape", size, density, encoded)
		}
	}
	for _, v := range []string{"bm1.orders_2024", "bm1.my.table", "BM1.AAAA", "bm2.AAAA", "bm1.AA=A", "bm1." + strings.Repeat("A", MaxEncodedBodyLen+4)} {
		assert.False(t, validShape(v), v)
	}
}

package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

// acceleratorPrefixes maps known accelerator types to short human-readable prefixes.
var acceleratorPrefixes = map[string]string{
	"GPU_1xA10":  "gpu-a10",
	"GPU_8xH100": "gpu-h100",
}

// GenerateSessionName creates a human-readable session name from the accelerator type.
// Format: <prefix>-<random_hex>, e.g. "gpu-a10-f3a2b1c0".
func GenerateSessionName(accelerator string) string {
	prefix, ok := acceleratorPrefixes[accelerator]
	if !ok {
		prefix = strings.ToLower(strings.ReplaceAll(accelerator, "_", "-"))
	}

	date := time.Now().Format("20060102")
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "databricks-" + prefix + "-" + date + "-" + hex.EncodeToString(b)
}

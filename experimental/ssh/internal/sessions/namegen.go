package sessions

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// acceleratorPrefixes maps known accelerator types to short human-readable prefixes.
var acceleratorPrefixes = map[string]string{
	"GPU_1xA10":  "gpu-a10",
	"GPU_8xH100": "gpu-h100",
}

// GenerateSessionName creates a human-readable session name from the accelerator type
// and workspace host. The workspace host is hashed into the name to avoid SSH known_hosts
// conflicts when connecting to different workspaces.
// Format: databricks-<prefix>-<date>-<workspace_hash><random_hex>.
func GenerateSessionName(accelerator, workspaceHost string) string {
	prefix, ok := acceleratorPrefixes[accelerator]
	if !ok {
		prefix = strings.ToLower(strings.ReplaceAll(accelerator, "_", "-"))
	}

	date := time.Now().Format("20060102")

	// Include a short hash of the workspace host to avoid known_hosts conflicts
	// when connecting to different workspaces.
	wsHash := md5.Sum([]byte(workspaceHost))
	wsHashStr := hex.EncodeToString(wsHash[:])[:4]

	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return "databricks-" + prefix + "-" + date + "-" + wsHashStr + hex.EncodeToString(b)
}

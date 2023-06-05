package fs

import (
	"fmt"
	"strings"
)

func resolveDbfsPath(path string) (string, error) {
	if !strings.HasPrefix(path, "dbfs:/") {
		return "", fmt.Errorf("expected dbfs path (with the dbfs:/ prefix): %s", path)
	}

	return strings.TrimPrefix(path, "dbfs:"), nil
}

package completion

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var multiBlankLine = regexp.MustCompile(`\n{3,}`)

// Uninstall removes shell completion config. Returns the file path that was
// modified and whether it was actually installed.
func Uninstall(shell Shell, homeDir string) (filePath string, wasInstalled bool, err error) {
	filePath = TargetFilePath(shell, homeDir)

	if shell == Fish {
		return uninstallFish(filePath)
	}
	return uninstallRC(filePath)
}

// uninstallFish handles the file-drop model: remove the file only if it
// contains our marker. This avoids deleting completions installed by a package
// manager or created by the user.
func uninstallFish(filePath string) (string, bool, error) {
	content, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return filePath, false, nil
	}
	if err != nil {
		return filePath, false, err
	}

	if !strings.Contains(string(content), BeginMarker) {
		return filePath, false, nil
	}

	if err := os.Remove(filePath); err != nil {
		return filePath, false, err
	}
	return filePath, true, nil
}

// uninstallRC handles the RC file model: find and remove the marker block.
func uninstallRC(filePath string) (string, bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return filePath, false, nil
	}
	if err != nil {
		return filePath, false, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return filePath, false, err
	}

	text := string(content)
	beginIdx := strings.Index(text, BeginMarker)
	if beginIdx == -1 {
		return filePath, false, nil
	}

	// Find the line number of BEGIN for error reporting.
	beginLine := strings.Count(text[:beginIdx], "\n") + 1

	// Look for END marker after BEGIN.
	afterBegin := text[beginIdx:]
	endIdx := strings.Index(afterBegin, EndMarker)
	if endIdx == -1 {
		return filePath, false, fmt.Errorf(
			"found corrupted completion block in %s: missing end marker. Please remove the block starting at line %d manually",
			filePath, beginLine,
		)
	}

	// Calculate absolute end position (after the END marker line including newline).
	blockEnd := beginIdx + endIdx + len(EndMarker)
	if blockEnd < len(text) && text[blockEnd] == '\n' {
		blockEnd++
	}

	result := text[:beginIdx] + text[blockEnd:]

	// Collapse double blank lines left by removal.
	result = multiBlankLine.ReplaceAllString(result, "\n\n")

	return filePath, true, os.WriteFile(filePath, []byte(result), info.Mode())
}

package textutil

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// We leave the full range of unicode letters in tact, but remove all "special" characters,
// including spaces and dots, which are not supported in e.g. experiment names or YAML keys.
func NormalizeString(name string) string {
	name = strings.ToLower(name)
	s := strings.Map(replaceNonAlphanumeric, name)

	// Replacing multiple underscores with a single one.
	re := regexp.MustCompile(`_+`)
	s = re.ReplaceAllString(s, "_")

	// Removing leading and trailing underscores.
	return strings.Trim(s, "_")
}

func replaceNonAlphanumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	return '_'
}

// NormalizePathComponent creates a filesystem-safe filename by filtering out
// characters that are not allowed in filenames, especially on Windows.
// Windows-incompatible characters: < > : " | ? * \ /
// It also handles reserved Windows filenames and ensures the result
// is not empty, properly trimmed, and within length limits.
// See https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
func NormalizePathComponent(name string) string {
	// Filter out filesystem-incompatible characters.
	s := strings.Map(filterFilesystemIncompatible, name)

	// Check for reserved Windows filenames (case-insensitive)
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	// Remove file extension for comparison
	baseName := s
	if dotIndex := strings.LastIndex(s, "."); dotIndex != -1 {
		baseName = s[:dotIndex]
	}

	// Check if the base name matches any reserved name (case-insensitive)
	for _, reserved := range reservedNames {
		if strings.EqualFold(baseName, reserved) {
			// If it's a reserved name, append underscore to make it valid
			if dotIndex := strings.LastIndex(s, "."); dotIndex != -1 {
				// Has extension: insert underscore before extension
				s = s[:dotIndex] + "_" + s[dotIndex:]
			} else {
				// No extension: append underscore
				s = s + "_"
			}
			break
		}
	}

	// Remove trailing spaces and periods from the filename
	s = strings.TrimRight(s, " .")

	// Ensure the filename is not empty after filtering.
	if s == "" {
		s = "untitled"
	}

	// Windows has a 255 character limit for filenames, so truncate if necessary.
	if len(s) > 255 {
		s = s[:255]
	}

	return s
}

var incompatibleRunes = []rune{'<', '>', ':', '"', '|', '?', '*', '\\', '/'}

func filterFilesystemIncompatible(r rune) rune {
	// Reject Windows-incompatible characters by returning -1 (which removes them).
	if slices.Contains(incompatibleRunes, r) {
		return '_'
	}

	// Keep all other characters.
	return r
}

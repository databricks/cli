package patchwheel

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	versionKey = "Version:"
	nameKey    = "Name:"
)

// readMetadataAndRecord scans the zip file for files matching the patterns
// "*.dist-info/METADATA" and "*.dist-info/RECORD". If multiple are found, it picks
// the first one encountered.
func readMetadataAndRecord(r *zip.ReadCloser) (metadataFile, recordFile *zip.File, oldDistInfoPrefix string, err error) {
	for _, f := range r.File {
		// Use filepath.Match to filter files in a .dist-info directory.
		if matched, _ := filepath.Match("*.dist-info/METADATA", f.Name); matched {
			if metadataFile == nil {
				metadataFile = f
				// Determine the old dist-info directory prefix.
				if i := strings.LastIndex(f.Name, "/"); i != -1 {
					oldDistInfoPrefix = f.Name[:i+1]
				}
			}
		} else if matched, _ := filepath.Match("*.dist-info/RECORD", f.Name); matched {
			if recordFile == nil {
				recordFile = f
			}
		}
	}
	if metadataFile == nil || recordFile == nil {
		return nil, nil, "", errors.New("wheel missing required METADATA or RECORD")
	}
	return metadataFile, recordFile, oldDistInfoPrefix, nil
}

// parseMetadata scans the METADATA content for the "Version:" and "Name:" fields.
func parseMetadata(content []byte) (version, distribution string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, versionKey) {
			version = strings.TrimSpace(strings.TrimPrefix(line, versionKey))
		} else if strings.HasPrefix(line, nameKey) {
			distribution = strings.TrimSpace(strings.TrimPrefix(line, nameKey))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if version == "" || distribution == "" {
		return "", "", errors.New("could not parse METADATA for version or distribution")
	}
	return version, distribution, nil
}

// patchMetadata returns new METADATA content with an updated "Version:" field.
func patchMetadata(content []byte, newVersion string) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Version:") {
			line = "Version: " + newVersion
		}
		buf.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// patchRecord updates RECORD content: it replaces the old dist-info prefix with the new one
// in all file paths and, for the METADATA entry, updates the hash and size.
func patchRecord(recordContent []byte, oldDistInfoPrefix, newDistInfoPrefix, metadataHash string, metadataSize int) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(recordContent))
	var newLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			newLines = append(newLines, line)
			continue
		}
		origPath := parts[0]
		if strings.HasPrefix(origPath, oldDistInfoPrefix) {
			parts[0] = newDistInfoPrefix + origPath[len(oldDistInfoPrefix):]
		}
		// For the METADATA file entry, update hash and size.
		if strings.HasSuffix(parts[0], "METADATA") {
			parts[1] = "sha256=" + metadataHash
			parts[2] = strconv.Itoa(metadataSize)
		}
		newLines = append(newLines, strings.Join(parts, ","))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return []byte(strings.Join(newLines, "\n") + "\n"), nil
}

func readFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// PatchWheel patches a Python wheel file by updating its version in METADATA and RECORD.
// It returns the path to the new wheel.
// The function is idempotent: repeated calls with the same input will produce the same output.
// If the target wheel already exists, it returns the path to the existing wheel without processing.
func PatchWheel(ctx context.Context, path, outputDir string) (string, error) {
	// Get the modification time of the input wheel
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	wheelMtime := fileInfo.ModTime().UTC()

	// Parse the wheel filename to extract components
	wheelInfo, err := ParseWheelFilename(path)
	if err != nil {
		return "", err
	}

	// Get the base version without any local version
	baseVersion := strings.SplitN(wheelInfo.Version, "+", 2)[0]

	// Calculate the timestamp suffix for the new version
	dt := strings.Replace(wheelMtime.Format("20060102150405.00"), ".", "", 1)
	dt = strings.Replace(dt, ".", "", 1)
	newVersion := baseVersion + "+" + dt

	// Create the new wheel filename
	newFilename := fmt.Sprintf("%s-%s-%s.whl",
		wheelInfo.Distribution,
		newVersion,
		strings.Join(wheelInfo.Tags, "-"))
	outpath := filepath.Join(outputDir, newFilename)

	// Check if the target wheel already exists
	if _, err := os.Stat(outpath); err == nil {
		// Target wheel already exists, return its path
		return outpath, nil
	}

	// Target wheel doesn't exist, proceed with patching
	// Create a temporary file in the same directory with a unique name
	tmpFile := outpath + fmt.Sprintf(".tmp%d", os.Getpid())
	
	// Ensure the temporary file is removed if we exit early
	defer func() {
		if _, statErr := os.Stat(tmpFile); statErr == nil {
			os.Remove(tmpFile)
		}
	}()
	
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	metadataFile, recordFile, oldDistInfoPrefix, err := readMetadataAndRecord(r)
	if err != nil {
		return "", err
	}

	metadataContent, err := readFile(metadataFile)
	if err != nil {
		return "", err
	}

	recordContent, err := readFile(recordFile)
	if err != nil {
		return "", err
	}

	// Verify the metadata version and distribution match what we extracted from filename
	metadataVersion, metadataDistribution, err := parseMetadata(metadataContent)
	if err != nil {
		return "", err
	}

	// Verify that the distribution name in the metadata matches the one from the filename
	if metadataDistribution != wheelInfo.Distribution {
		return "", fmt.Errorf("distribution name mismatch: %s (metadata) vs %s (filename)",
			metadataDistribution, wheelInfo.Distribution)
	}

	// Verify that the base version in the metadata matches the one from the filename
	metadataBaseVersion := strings.SplitN(metadataVersion, "+", 2)[0]
	if metadataBaseVersion != baseVersion {
		return "", fmt.Errorf("version mismatch: %s (metadata) vs %s (filename)",
			metadataBaseVersion, baseVersion)
	}

	// log.Infof(ctx, "path=%s version=%s newVersion=%s distribution=%s", path, metadataVersion, newVersion, metadataDistribution)

	// Patch the METADATA content.
	newMetadata, err := patchMetadata(metadataContent, newVersion)
	if err != nil {
		return "", err
	}

	// Compute the new hash for METADATA.
	h := sha256.New()
	h.Write(newMetadata)
	metadataHash := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
	metadataSize := len(newMetadata)

	// Compute the new dist-info directory prefix.
	newDistInfoPrefix := ""
	if idx := strings.LastIndex(oldDistInfoPrefix, "-"); idx != -1 {
		base := oldDistInfoPrefix[:idx]
		newDistInfoPrefix = base + "-" + newVersion + ".dist-info/"
	} else {
		return "", fmt.Errorf("unexpected dist-info directory format: %s", oldDistInfoPrefix)
	}

	// Patch the RECORD content.
	newRecord, err := patchRecord(recordContent, oldDistInfoPrefix, newDistInfoPrefix, metadataHash, metadataSize)
	if err != nil {
		return "", err
	}

	// Write to the temporary file first
	outFile, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	// Write a new wheel (zip archive) with the patched files.
	zipw := zip.NewWriter(outFile)
	for _, f := range r.File {
		// If the file is inside the old dist-info directory, update its name.
		newName := f.Name
		if strings.HasPrefix(f.Name, oldDistInfoPrefix) {
			newName = newDistInfoPrefix + f.Name[len(oldDistInfoPrefix):]
		}
		header := &zip.FileHeader{
			Name:   newName,
			Method: f.Method,
		}

		header.Modified = f.ModTime()
		header.Comment = f.Comment
		if f.FileInfo().IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}

		writer, err := zipw.CreateHeader(header)
		if err != nil {
			return "", err
		}

		// For METADATA and RECORD files, write the modified content.
		if strings.HasSuffix(f.Name, "METADATA") && strings.HasPrefix(f.Name, oldDistInfoPrefix) {
			_, err = writer.Write(newMetadata)
			if err != nil {
				return "", err
			}
		} else if strings.HasSuffix(f.Name, "RECORD") && strings.HasPrefix(f.Name, oldDistInfoPrefix) {
			_, err = writer.Write(newRecord)
			if err != nil {
				return "", err
			}
		} else {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			_, err = io.Copy(writer, rc)
			if err != nil {
				rc.Close()
				return "", err
			}
			if err := rc.Close(); err != nil {
				return "", err
			}
		}
	}

	if err := zipw.Close(); err != nil {
		return "", err
	}
	
	// Close the file before renaming
	outFile.Close()
	
	// Rename the temporary file to the final output path (atomic operation)
	if err := os.Rename(tmpFile, outpath); err != nil {
		return "", err
	}

	return outpath, nil
}

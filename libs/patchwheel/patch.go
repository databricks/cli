package patchwheel

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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
func readMetadataAndRecord(r *zip.ReadCloser) (metadataFile, recordFile *zip.File, oldDistInfoPrefix string) {
	for _, f := range r.File {
		if metadataFile == nil {
			matched, _ := filepath.Match("*.dist-info/METADATA", f.Name)
			if matched {
				metadataFile = f
				// Determine the old dist-info directory prefix.
				if i := strings.LastIndex(f.Name, "/"); i != -1 {
					oldDistInfoPrefix = f.Name[:i+1]
				}

				if recordFile != nil {
					break
				}

				continue
			}
		}

		if recordFile == nil {
			matched, _ := filepath.Match("*.dist-info/RECORD", f.Name)
			if matched {
				recordFile = f

				if metadataFile != nil {
					break
				}
			}
		}
	}

	return metadataFile, recordFile, oldDistInfoPrefix
}

// patchMetadata returns new METADATA content with an updated "Version:" field and validates that previous version matches oldVersion
func patchMetadata(r io.Reader, oldVersion, newVersion string) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, versionKey) {
			foundVersion := strings.TrimSpace(line[len(versionKey):])
			if foundVersion != oldVersion {
				return nil, fmt.Errorf("Unexpected version in METADATA: %s (expected %s)", strings.TrimSpace(line), oldVersion)
			}
			line = versionKey + newVersion
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
func patchRecord(r io.Reader, oldDistInfoPrefix, newDistInfoPrefix, metadataHash string, metadataSize int) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	var newLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			// If the line doesn't have enough parts, preserve it as-is
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

// PatchWheel patches a Python wheel file by updating its version in METADATA and RECORD.
// It returns the path to the new wheel.
// The function is idempotent: repeated calls with the same input will produce the same output.
// If the target wheel already exists, it returns the path to the existing wheel without processing.
func PatchWheel(ctx context.Context, path, outputDir string) (string, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	wheelMtime := fileInfo.ModTime().UTC()

	filename := filepath.Base(path)
	wheelInfo, err := ParseWheelFilename(filename)
	if err != nil {
		return "", err
	}

	baseVersion := strings.SplitN(wheelInfo.Version, "+", 2)[0]

	dt := strings.Replace(wheelMtime.Format("20060102150405.00"), ".", "", 1)
	dt = strings.Replace(dt, ".", "", 1)
	newVersion := baseVersion + "+" + dt

	newFilename := fmt.Sprintf("%s-%s-%s.whl",
		wheelInfo.Distribution,
		newVersion,
		strings.Join(wheelInfo.Tags, "-"))
	outpath := filepath.Join(outputDir, newFilename)

	if _, err := os.Stat(outpath); err == nil {
		// Target wheel already exists, return its path
		return outpath, nil
	}

	// Target wheel doesn't exist, proceed with patching
	// Create a temporary file in the same directory with a unique name
	tmpFile := outpath + fmt.Sprintf(".tmp%d", os.Getpid())

	needRemoval := true

	defer func() {
		if needRemoval {
			_ = os.Remove(tmpFile)
		}
	}()

	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	metadataFile, recordFile, oldDistInfoPrefix := readMetadataAndRecord(r)
	if metadataFile == nil {
		return "", fmt.Errorf("wheel %s missing METADATA file", path)
	}

	if recordFile == nil {
		return "", fmt.Errorf("wheel %s missing RECORD file", path)
	}

	metadataReader, err := metadataFile.Open()
	if err != nil {
		return "", err
	}
	defer metadataReader.Close()

	recordReader, err := recordFile.Open()
	if err != nil {
		return "", err
	}
	defer recordReader.Close()

	newMetadata, err := patchMetadata(metadataReader, wheelInfo.Version, newVersion)
	if err != nil {
		return "", err
	}

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

	newRecord, err := patchRecord(recordReader, oldDistInfoPrefix, newDistInfoPrefix, metadataHash, metadataSize)
	if err != nil {
		return "", err
	}

	outFile, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

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

	outFile.Close()

	if err := os.Rename(tmpFile, outpath); err != nil {
		return "", err
	}

	needRemoval = false
	return outpath, nil
}

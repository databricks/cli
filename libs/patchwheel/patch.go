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

func findMetadataAndRecord(r *zip.ReadCloser, oldDistInfoPrefix string) (metadataFile, recordFile *zip.File) {
	for _, f := range r.File {
		if metadataFile == nil && f.Name == oldDistInfoPrefix+"METADATA" {
			metadataFile = f
		}

		if recordFile == nil && f.Name == oldDistInfoPrefix+"RECORD" {
			recordFile = f

			if metadataFile != nil {
				break
			}
		}
	}

	return metadataFile, recordFile
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
// The version is updated according to the following rules:
//   - if there is an existing part after + it is dropped
//   - append +<mtime of the original wheel> to version
//
// The function is idempotent: repeated calls with the same input will produce the same output.
// If the target wheel already exists, it returns the path to the existing wheel without redoing the patching.
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

	newVersion, newFilename := CalculateNewVersion(wheelInfo, wheelMtime)
	outpath := filepath.Join(outputDir, newFilename)

	if _, err := os.Stat(outpath); err == nil {
		// Target wheel already exists, return its path
		return outpath, nil
	}

	tmpFilename := outpath + fmt.Sprintf(".tmp%d", os.Getpid())

	needRemoval := true

	defer func() {
		if needRemoval {
			_ = os.Remove(tmpFilename)
		}
	}()

	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	oldDistInfoPrefix := wheelInfo.Distribution + "-" + wheelInfo.Version + ".dist-info/"
	metadataFile, recordFile := findMetadataAndRecord(r, oldDistInfoPrefix)
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

	newMetadata, err := patchMetadata(metadataReader, wheelInfo.Version, newVersion)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(newMetadata)
	metadataHash := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
	metadataSize := len(newMetadata)

	// Compute the new dist-info directory prefix.
	newDistInfoPrefix := strings.Replace(oldDistInfoPrefix, wheelInfo.Version, newVersion, 1)
	if newDistInfoPrefix == oldDistInfoPrefix {
		return "", fmt.Errorf("unexpected dist-info directory format: %s (version=%s)", oldDistInfoPrefix, wheelInfo.Version)
	}

	recordReader, err := recordFile.Open()
	if err != nil {
		return "", err
	}
	defer recordReader.Close()

	newRecord, err := patchRecord(recordReader, oldDistInfoPrefix, newDistInfoPrefix, metadataHash, metadataSize)
	if err != nil {
		return "", err
	}

	outFile, err := os.Create(tmpFilename)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	metadataUpdated := 0
	recordUpdated := 0

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

		if f.Name == metadataFile.Name {
			_, err = writer.Write(newMetadata)
			if err != nil {
				return "", err
			}
			metadataUpdated += 1
		} else if f.Name == recordFile.Name {
			_, err = writer.Write(newRecord)
			if err != nil {
				return "", err
			}
			recordUpdated += 1
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

	if metadataUpdated != 1 {
		return "", errors.New("Could not update METADATA")
	}

	if recordUpdated != 1 {
		return "", errors.New("Could not update RECORD")
	}

	if err := os.Rename(tmpFilename, outpath); err != nil {
		return "", err
	}
	needRemoval = false

	return outpath, nil
}

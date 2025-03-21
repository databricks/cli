package patchwheel

import (
	"archive/zip"
	"bufio"
	"bytes"
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

var versionKey []byte = []byte("Version:")

func findFile(r *zip.ReadCloser, filename string) *zip.File {
	for _, f := range r.File {
		if f.Name == filename {
			return f
		}
	}
	return nil
}

// patchMetadata returns new METADATA content with an updated "Version:" field and validates that previous version matches oldVersion
func patchMetadata(r io.Reader, oldVersion, newVersion string) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		if versionValue, ok := bytes.CutPrefix(line, versionKey); ok {
			foundVersion := string(bytes.TrimSpace(versionValue))
			if foundVersion != oldVersion {
				return nil, fmt.Errorf("Unexpected version in METADATA: %s (expected %s)", strings.TrimSpace(string(line)), oldVersion)
			}
			buf.WriteString(string(versionKey) + " " + newVersion + "\n")
		} else {
			buf.Write(line)
			buf.WriteString("\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// patchRecord updates RECORD content: it replaces the old dist-info prefix with the new one
// in all file paths and, for the METADATA entry, updates the hash and size.
func patchRecord(r io.Reader, oldDistInfoPrefix, newDistInfoPrefix, metadataHash string, metadataSize int) ([]byte, error) {
	metadataPath := newDistInfoPrefix + "METADATA"
	scanner := bufio.NewScanner(r)
	var buf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		parts := strings.Split(string(line), ",")

		if len(parts) < 3 {
			// If the line doesn't have enough parts, preserve it as-is
			buf.Write(line)
			buf.WriteString("\n")
			continue
		}

		origPath := parts[0]
		pathSuffix, hasDistPrefix := strings.CutPrefix(origPath, oldDistInfoPrefix)
		if hasDistPrefix {
			parts[0] = newDistInfoPrefix + pathSuffix
		}

		if metadataPath == parts[0] {
			parts[1] = "sha256=" + metadataHash
			parts[2] = strconv.Itoa(metadataSize)
		}

		buf.WriteString(strings.Join(parts, ",") + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PatchWheel reads an existing wheel file path and outputs a new one in outputDir,
// with a version modified according to the following rules:
//   - if there is an existing part after + it is dropped
//   - append +<mtime of the original wheel> to version
//
// All parts of wheel are modified to ensure the wheel is in correct format:
// METADATA: Version field is updated
// RECORD: METADATA entry is updated with correct hash and size
// <dist>-<version>.dist-info directory is renamed to <dist>-<newVersion>.dist-info
//
// The function is idempotent: repeated calls with the same input will produce the same output.
// If the target wheel already exists, it returns the path to the existing wheel without redoing the patching.
func PatchWheel(path, outputDir string) (string, bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", false, err
	}
	wheelMtime := fileInfo.ModTime().UTC()

	filename := filepath.Base(path)
	wheelInfo, err := ParseWheelFilename(filename)
	if err != nil {
		return "", false, err
	}

	newVersion, newFilename := calculateNewVersion(wheelInfo, wheelMtime)
	outpath := filepath.Join(outputDir, newFilename)

	if _, err := os.Stat(outpath); err == nil {
		return outpath, false, nil
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
		return "", false, err
	}
	defer r.Close()

	oldDistInfoPrefix := wheelInfo.Distribution + "-" + wheelInfo.Version + ".dist-info/"
	metadataFile := findFile(r, oldDistInfoPrefix+"METADATA")
	if metadataFile == nil {
		return "", false, fmt.Errorf("wheel %s missing %sMETADATA", path, oldDistInfoPrefix)
	}

	recordFile := findFile(r, oldDistInfoPrefix+"RECORD")
	if recordFile == nil {
		return "", false, fmt.Errorf("wheel %s missing %sRECORD file", path, oldDistInfoPrefix)
	}

	metadataReader, err := metadataFile.Open()
	if err != nil {
		return "", false, err
	}
	defer metadataReader.Close()

	newMetadata, err := patchMetadata(metadataReader, wheelInfo.Version, newVersion)
	if err != nil {
		return "", false, err
	}

	h := sha256.New()
	h.Write(newMetadata)
	metadataHash := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
	metadataSize := len(newMetadata)

	// Compute the new dist-info directory prefix.
	newDistInfoPrefix := strings.Replace(oldDistInfoPrefix, wheelInfo.Version, newVersion, 1)
	if newDistInfoPrefix == oldDistInfoPrefix {
		return "", false, fmt.Errorf("unexpected dist-info directory format: %s (version=%s)", oldDistInfoPrefix, wheelInfo.Version)
	}

	recordReader, err := recordFile.Open()
	if err != nil {
		return "", false, err
	}
	defer recordReader.Close()

	newRecord, err := patchRecord(recordReader, oldDistInfoPrefix, newDistInfoPrefix, metadataHash, metadataSize)
	if err != nil {
		return "", false, err
	}

	outFile, err := os.Create(tmpFilename)
	if err != nil {
		return "", false, err
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
			return "", false, err
		}

		if f.Name == metadataFile.Name {
			_, err = writer.Write(newMetadata)
			if err != nil {
				return "", false, err
			}
			metadataUpdated += 1
		} else if f.Name == recordFile.Name {
			_, err = writer.Write(newRecord)
			if err != nil {
				return "", false, err
			}
			recordUpdated += 1
		} else {
			rc, err := f.Open()
			if err != nil {
				return "", false, err
			}
			_, err = io.Copy(writer, rc)
			if err != nil {
				rc.Close()
				return "", false, err
			}
			if err := rc.Close(); err != nil {
				return "", false, err
			}
		}
	}

	if err := zipw.Close(); err != nil {
		return "", false, err
	}

	outFile.Close()

	if metadataUpdated != 1 {
		return "", false, errors.New("Could not update METADATA")
	}

	if recordUpdated != 1 {
		return "", false, errors.New("Could not update RECORD")
	}

	if err := os.Rename(tmpFilename, outpath); err != nil {
		return "", false, err
	}
	needRemoval = false

	return outpath, true, nil
}

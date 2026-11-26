package patchwheel

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
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
	// METADATA is an email-style message: the headers end at the first empty line and the
	// rest is the description, which may contain lines that look like headers.
	inHeaders := true
	replaced := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			inHeaders = false
		}
		if versionValue, ok := bytes.CutPrefix(line, versionKey); ok && inHeaders {
			foundVersion := string(bytes.TrimSpace(versionValue))
			if foundVersion != oldVersion {
				return nil, fmt.Errorf("unexpected version in METADATA: %s (expected %s)", strings.TrimSpace(string(line)), oldVersion)
			}
			buf.WriteString(string(versionKey) + " " + newVersion + "\n")
			replaced++
		} else {
			buf.Write(line)
			buf.WriteString("\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if replaced != 1 {
		return nil, fmt.Errorf("expected one Version header in METADATA, found %d", replaced)
	}
	return buf.Bytes(), nil
}

// patchRecord updates RECORD content: it renames every path with renamePath and, for the
// METADATA entry (metadataPath, after renaming), updates the hash and size.
func patchRecord(r io.Reader, renamePath func(string) string, metadataPath, metadataHash string, metadataSize int) ([]byte, error) {
	// RECORD is CSV: a path that contains a comma or a quote is quoted, so it must be
	// parsed as CSV for renamePath to see the real path.
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(row) == 1 && strings.TrimSpace(row[0]) == "" {
			continue
		}

		// Rows without path, hash and size are kept as they are.
		if len(row) >= 3 {
			row[0] = renamePath(row[0])
			if metadataPath == row[0] {
				row[1] = "sha256=" + metadataHash
				row[2] = strconv.Itoa(metadataSize)
			}
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
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
// <dist>-<version>.data directory, if present, is renamed to <dist>-<newVersion>.data
// File modes of all entries are preserved.
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

	newDistInfoPrefix := wheelInfo.Distribution + "-" + newVersion + ".dist-info/"
	oldDataPrefix := wheelInfo.Distribution + "-" + wheelInfo.Version + ".data/"
	newDataPrefix := wheelInfo.Distribution + "-" + newVersion + ".data/"
	renamePath := func(name string) string {
		if suffix, ok := strings.CutPrefix(name, oldDistInfoPrefix); ok {
			return newDistInfoPrefix + suffix
		}
		if suffix, ok := strings.CutPrefix(name, oldDataPrefix); ok {
			return newDataPrefix + suffix
		}
		return name
	}

	recordReader, err := recordFile.Open()
	if err != nil {
		return "", false, err
	}
	defer recordReader.Close()

	newRecord, err := patchRecord(recordReader, renamePath, newDistInfoPrefix+"METADATA", metadataHash, metadataSize)
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
		header := &zip.FileHeader{
			Name:   renamePath(f.Name),
			Method: f.Method,
			// Installers (pip, uv) restore the executable bit from the unix mode in
			// ExternalAttrs; CreatorVersion says which OS the attributes are for.
			CreatorVersion: f.CreatorVersion,
			ExternalAttrs:  f.ExternalAttrs,
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

		switch f.Name {
		case metadataFile.Name:
			_, err = writer.Write(newMetadata)
			if err != nil {
				return "", false, err
			}
			metadataUpdated += 1
		case recordFile.Name:
			_, err = writer.Write(newRecord)
			if err != nil {
				return "", false, err
			}
			recordUpdated += 1
		default:
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
		return "", false, errors.New("could not update METADATA")
	}

	if recordUpdated != 1 {
		return "", false, errors.New("could not update RECORD")
	}

	if err := os.Rename(tmpFilename, outpath); err != nil {
		return "", false, err
	}
	needRemoval = false

	return outpath, true, nil
}

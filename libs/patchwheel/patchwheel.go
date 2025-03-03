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

const (
	versionKey = "Version:"
	nameKey    = "Name:"
)

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

// WheelInfo contains information extracted from a wheel filename
type WheelInfo struct {
	Distribution string   // Package distribution name
	Version      string   // Package version
	Tags         []string // Python tags (python_tag, abi_tag, platform_tag)
}

// ExtractVersionFromWheelFilename extracts the version from a wheel filename.
// Wheel filenames follow the pattern: {distribution}-{version}-{python_tag}-{abi_tag}-{platform_tag}.whl
func ExtractVersionFromWheelFilename(filename string) (string, error) {
	info, err := ParseWheelFilename(filename)
	if err != nil {
		return "", err
	}
	return info.Version, nil
}

// ParseWheelFilename parses a wheel filename and extracts its components.
// Wheel filenames follow the pattern: {distribution}-{version}-{python_tag}-{abi_tag}-{platform_tag}.whl
func ParseWheelFilename(filename string) (*WheelInfo, error) {
	base := filepath.Base(filename)
	parts := strings.Split(base, "-")
	if len(parts) < 5 || !strings.HasSuffix(parts[len(parts)-1], ".whl") {
		return nil, fmt.Errorf("invalid wheel filename format: %s", filename)
	}
	
	// The last three parts are always tags
	tagStartIdx := len(parts) - 3
	
	// Everything before the tags except the version is the distribution
	versionIdx := tagStartIdx - 1
	
	// Distribution may contain hyphens, so join all parts before the version
	distribution := strings.Join(parts[:versionIdx], "-")
	version := parts[versionIdx]
	
	// Extract tags (remove .whl from the last one)
	tags := make([]string, 3)
	copy(tags, parts[tagStartIdx:])
	tags[2] = strings.TrimSuffix(tags[2], ".whl")
	
	return &WheelInfo{
		Distribution: distribution,
		Version:      version,
		Tags:         tags,
	}, nil
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

	version, distribution, err := parseMetadata(metadataContent)
	if err != nil {
		return "", err
	}

	// If there's already a local version (after +), strip it off
	baseVersion := strings.SplitN(version, "+", 2)[0]

	// Use the wheel file's modification time for idempotency
	dt := strings.Replace(wheelMtime.Format("20060102150405.00"), ".", "", 1)
	dt = strings.Replace(dt, ".", "", 1)

	newVersion := baseVersion + "+" + dt
	// log.Infof(ctx, "path=%s version=%s newVersion=%s distribution=%s", path, version, newVersion, distribution)

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

	// Create the new wheel filename.
	newFilename := fmt.Sprintf("%s-%s-py3-none-any.whl", distribution, newVersion)
	outpath := filepath.Join(outputDir, newFilename)
	outFile, err := os.Create(outpath)
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
	return outpath, nil
}

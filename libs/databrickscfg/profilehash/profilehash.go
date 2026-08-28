package profilehash

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"maps"
	"slices"

	"github.com/databricks/databricks-sdk-go/config"
)

// Compute hashes parsed profile values so formatting-only edits do not
// invalidate credentials.
func Compute(values map[string]string) string {
	keys := slices.Sorted(maps.Keys(values))
	var serialized []byte

	for _, key := range keys {
		value := values[key]
		serialized = binary.AppendUvarint(serialized, uint64(len(key)))
		serialized = append(serialized, key...)
		serialized = binary.AppendUvarint(serialized, uint64(len(value)))
		serialized = append(serialized, value...)
	}

	sum := sha256.Sum256(serialized)

	return hex.EncodeToString(sum[:])
}

// FromFile hashes every parsed key and value in the named profile section.
func FromFile(configFilePath, profileName string) (string, error) {
	file, err := config.LoadFile(configFilePath)
	if err != nil {
		return "", err
	}

	section, err := file.GetSection(profileName)
	if err != nil {
		return "", err
	}

	return Compute(section.KeysHash()), nil
}

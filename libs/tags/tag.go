package tags

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// The tag type holds the validation and normalization rules for
// a cloud provider's resource tags as applied by Databricks.
type tag struct {
	keyLength    int
	keyPattern   *regexp.Regexp
	keyNormalize transformer

	valueLength    int
	valuePattern   *regexp.Regexp
	valueNormalize transformer
}

func (t *tag) ValidateKey(s string) error {
	if len(s) == 0 {
		return errors.New("key must not be empty")
	}
	if len(s) > t.keyLength {
		return fmt.Errorf("key length %d exceeds maximum of %d", len(s), t.keyLength)
	}
	if strings.ContainsFunc(s, func(r rune) bool { return !unicode.Is(latin1, r) }) {
		return errors.New("key contains non-latin1 characters")
	}
	if !t.keyPattern.MatchString(s) {
		return fmt.Errorf("key %q does not match pattern %q", s, t.keyPattern)
	}
	return nil
}

func (t *tag) ValidateValue(s string) error {
	if len(s) > t.valueLength {
		return fmt.Errorf("value length %d exceeds maximum of %d", len(s), t.valueLength)
	}
	if strings.ContainsFunc(s, func(r rune) bool { return !unicode.Is(latin1, r) }) {
		return errors.New("value contains non-latin1 characters")
	}
	if !t.valuePattern.MatchString(s) {
		return fmt.Errorf("value %q does not match pattern %q", s, t.valuePattern)
	}
	return nil
}

func (t *tag) NormalizeKey(s string) string {
	return t.keyNormalize.transform(s)
}

func (t *tag) NormalizeValue(s string) string {
	return t.valueNormalize.transform(s)
}

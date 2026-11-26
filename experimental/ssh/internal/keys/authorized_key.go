package keys

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/crypto/ssh"
)

var (
	errMultipleAuthorizedKeyLines = errors.New("it has more than one line")
	errAuthorizedKeyOptions       = errors.New("it sets authorized_keys options")
	errNotAPublicKey              = errors.New("it is not a valid SSH public key")
	errAuthorizedKeyCertificate   = errors.New("it is an SSH certificate")
)

// GetAuthorizedKey reads the client public key from the secret scope and returns it as a
// single canonical authorized_keys line ("<type> <base64>\n").
//
// The CLI stores this secret with ssh.MarshalAuthorizedKey, which writes one plain public key
// without options. As hardening of that secret format, a secret in any other form is rejected
// rather than rewritten: one that has more than one line, sets authorized_keys options, is not
// a valid SSH public key, or is an SSH certificate. The error tells the user how to recover.
func GetAuthorizedKey(ctx context.Context, client *databricks.WorkspaceClient, scope, key string) ([]byte, error) {
	raw, err := GetSecret(ctx, client, scope, key)
	if err != nil {
		return nil, err
	}
	authorizedKey, err := canonicalAuthorizedKey(raw)
	if err != nil {
		return nil, fmt.Errorf("secret %q of secret scope %q is not in the expected format because %w. "+
			"The Databricks CLI stores a single public key without options there. "+
			"Delete the secret scope with 'databricks secrets delete-scope %s', using the same profile you use for "+
			"'databricks ssh connect' (for example --profile <name>), and connect again; "+
			"if you cannot delete it, ask a workspace admin to remove it", key, scope, err, scope)
	}
	return authorizedKey, nil
}

// canonicalAuthorizedKey parses raw as exactly one authorized_keys entry without options that
// holds a plain public key (not a certificate) and returns it re-marshaled, which also drops
// any comment. Errors never quote raw: it is arbitrary text that would end up in a terminal or
// log.
func canonicalAuthorizedKey(raw []byte) ([]byte, error) {
	line := bytes.TrimSpace(raw)
	// Require the secret to be exactly one line. Checking the rest returned by
	// ParseAuthorizedKey is not enough: it silently skips comment lines and lines it cannot
	// parse before the first key.
	if bytes.ContainsAny(line, "\r\n") {
		return nil, errMultipleAuthorizedKeyLines
	}
	publicKey, _, options, _, err := ssh.ParseAuthorizedKey(line)
	if err != nil {
		// The parse error can quote parts of the key (e.g. an unknown algorithm name).
		return nil, errNotAPublicKey
	}
	if len(options) > 0 {
		return nil, errAuthorizedKeyOptions
	}
	// A certificate carries its own restrictions (critical options such as force-command and
	// source-address) inside the key blob, where the options check above does not see them.
	// The CLI stores a plain public key, so reject certificates outright.
	if _, ok := publicKey.(*ssh.Certificate); ok {
		return nil, errAuthorizedKeyCertificate
	}
	return ssh.MarshalAuthorizedKey(publicKey), nil
}

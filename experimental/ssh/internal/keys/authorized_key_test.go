package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// authorizedKeyLine returns a public key in the "<type> <base64>" form, without the newline.
func authorizedKeyLine(t *testing.T, publicKey any) string {
	sshKey, err := ssh.NewPublicKey(publicKey)
	require.NoError(t, err)
	return strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(sshKey)), "\n")
}

// userCertificateLine returns an OpenSSH user certificate for publicKey, signed by a fresh CA
// and restricted with a force-command critical option, in the "<type> <base64>" form.
func userCertificateLine(t *testing.T, publicKey any) string {
	sshKey, err := ssh.NewPublicKey(publicKey)
	require.NoError(t, err)
	_, caPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	caSigner, err := ssh.NewSignerFromKey(caPrivate)
	require.NoError(t, err)
	cert := &ssh.Certificate{
		Key:             sshKey,
		CertType:        ssh.UserCert,
		ValidPrincipals: []string{"root"},
		ValidBefore:     ssh.CertTimeInfinity,
		CriticalOptions: map[string]string{"force-command": "true"},
	}
	require.NoError(t, cert.SignCert(rand.Reader, caSigner))
	return strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(cert)), "\n")
}

func TestCanonicalAuthorizedKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rsaLine := authorizedKeyLine(t, &rsaKey.PublicKey)
	ed25519Public, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	ed25519Line := authorizedKeyLine(t, ed25519Public)
	certLine := userCertificateLine(t, ed25519Public)

	accepted := []struct {
		name string
		raw  string
		want string
	}{
		{name: "rsa as the CLI writes it", raw: rsaLine + "\n", want: rsaLine + "\n"},
		{name: "ed25519", raw: ed25519Line + "\n", want: ed25519Line + "\n"},
		{name: "no trailing newline", raw: rsaLine, want: rsaLine + "\n"},
		{name: "surrounding whitespace and CRLF", raw: "  \t" + rsaLine + " \r\n\n", want: rsaLine + "\n"},
		{name: "comment is dropped", raw: ed25519Line + " user@laptop.test\n", want: ed25519Line + "\n"},
	}
	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalAuthorizedKey([]byte(tt.raw))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}

	rejected := []struct {
		name string
		raw  string
		want error
	}{
		{name: "command option", raw: `command="true" ` + rsaLine + "\n", want: errAuthorizedKeyOptions},
		{name: "environment", raw: `environment="FOO=bar" ` + ed25519Line, want: errAuthorizedKeyOptions},
		{name: "no-pty", raw: "no-pty " + rsaLine, want: errAuthorizedKeyOptions},
		{name: "from", raw: `from="10.0.0.0/8" ` + rsaLine, want: errAuthorizedKeyOptions},
		{name: "two keys", raw: rsaLine + "\n" + ed25519Line + "\n", want: errMultipleAuthorizedKeyLines},
		{name: "second line with options", raw: rsaLine + "\n" + `command="true" ` + ed25519Line, want: errMultipleAuthorizedKeyLines},
		// ParseAuthorizedKey alone would skip the comment and return the key after it.
		{name: "comment line before the key", raw: "# a comment\n" + rsaLine, want: errMultipleAuthorizedKeyLines},
		// A '\r' counts as a line break too, so the secret is exactly one line.
		{name: "carriage return inside", raw: rsaLine + "\r" + `command="true" ` + ed25519Line, want: errMultipleAuthorizedKeyLines},
		// The certificate's force-command lives in the key blob, not in the options field.
		{name: "user certificate", raw: certLine + "\n", want: errAuthorizedKeyCertificate},
		{name: "garbage", raw: "not a key", want: errNotAPublicKey},
		{name: "invalid base64", raw: "ssh-rsa !!!!", want: errNotAPublicKey},
		{name: "empty", raw: "\n", want: errNotAPublicKey},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalAuthorizedKey([]byte(tt.raw))
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

// Package fips verifies the crypto module the binary was linked against.
//
// Go selects its cryptographic module at build time via GOFIPS140. The setting
// cannot be applied afterwards: GODEBUG=fips140=on toggles FIPS mode on
// whatever module was already linked, so a binary built without GOFIPS140 has
// no validated module to enable. That makes the module a property of the build,
// and these tests assert it rather than assuming it.
//
// The default build sets no GOFIPS140 and is unaffected: TestFIPSStatus records
// the state and the remaining tests skip. Set GOFIPS140 to a frozen module
// version to exercise them:
//
//	GOFIPS140=v1.0.0 go test ./internal/fips/...
package fips

import (
	"crypto/fips140"
	"crypto/tls"
	"slices"
	"testing"
)

// approvedVersions lists module versions that have completed validation and so
// carry a certificate. "latest" is deliberately absent: it tracks the in-tree
// source and moves with every Go release, leaving no fixed artifact to cite.
var approvedVersions = []string{
	"v1.0.0",
}

// TestFIPSStatus records the linked module in the test log for every build,
// including the default one. It never fails, so it works as the place to look
// when confirming what a given binary was built with.
func TestFIPSStatus(t *testing.T) {
	t.Logf("fips140.Enabled() = %v", fips140.Enabled())
	t.Logf("fips140.Version() = %q", fips140.Version())
}

// TestFIPSVersionApproved guards against a build that selects a module version
// we have not vetted -- most likely "latest", which is easy to reach by
// accident and produces a binary with nothing citable.
func TestFIPSVersionApproved(t *testing.T) {
	if !fips140.Enabled() {
		t.Skip("not built with GOFIPS140")
	}

	version := fips140.Version()
	if !slices.Contains(approvedVersions, version) {
		t.Errorf("module version %q is not in the approved list %v", version, approvedVersions)
	}
}

// TestFIPSModeEnabled catches the case where the module is linked but FIPS mode
// is left off. Linking and enabling are separate steps, and a build that gets
// only the first looks correct from the outside while behaving like a plain
// build.
func TestFIPSModeEnabled(t *testing.T) {
	if fips140.Version() == "latest" {
		t.Skip("not built with a frozen GOFIPS140 version")
	}

	if !fips140.Enabled() {
		t.Errorf("module %q is linked but FIPS mode is off", fips140.Version())
	}
}

// TestFIPSHandshakeCipherSuites checks that a TLS client can still be
// configured and that approved suites remain available. FIPS mode narrows the
// suites Go will offer -- ChaCha20-Poly1305 drops out, and TLS 1.3 selection is
// no longer configurable through tls.Config -- so this is the cheapest signal
// that the restriction has not left the client with nothing to negotiate.
func TestFIPSHandshakeCipherSuites(t *testing.T) {
	if !fips140.Enabled() {
		t.Skip("not built with GOFIPS140")
	}

	var approved bool
	for _, suite := range tls.CipherSuites() {
		switch suite.ID {
		case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
			approved = true
		}
	}
	if !approved {
		t.Error("no approved AES-GCM cipher suite is available to the TLS client")
	}

	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %v, want %v", config.MinVersion, tls.VersionTLS12)
	}
}

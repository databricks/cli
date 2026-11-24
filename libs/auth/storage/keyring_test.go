package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// fakeBackend is a test double for keyringBackend. It records Set/Get/Delete
// calls and lets tests program responses.
type fakeBackend struct {
	items map[string]string // key = service+":"+account

	setErr    error
	getErr    error
	deleteErr error

	setBlock    bool // if true, Set blocks forever (for timeout tests)
	getBlock    bool
	deleteBlock bool

	// errAccount limits the errors and blocks above to accounts that contain
	// it, such as AccessTokenPartMarker for the access token parts.
	errAccount string

	// secretTooBig, when set, mirrors an OS backend's size check: Set rejects
	// the secret with keyring.ErrSetDataTooBig before storing anything.
	secretTooBig func(service, account, secret string) bool

	// afterGet, when set, runs after each Get has read the item, so that a
	// test can change the items between two reads as another process would.
	afterGet func(account string)
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{items: map[string]string{}}
}

func itemKey(service, account string) string { return service + ":" + account }

// accounts returns the sorted accounts of the items in the databricks-cli
// service.
func (f *fakeBackend) accounts() []string {
	var accounts []string
	for k := range f.items {
		if account, ok := strings.CutPrefix(k, itemKey(keyringServiceName, "")); ok {
			accounts = append(accounts, account)
		}
	}
	slices.Sort(accounts)
	return accounts
}

func (f *fakeBackend) Set(service, account, secret string) error {
	matches := strings.Contains(account, f.errAccount)
	if f.setBlock && matches {
		select {}
	}
	if f.secretTooBig != nil && f.secretTooBig(service, account, secret) {
		return keyring.ErrSetDataTooBig
	}
	if f.setErr != nil && matches {
		return f.setErr
	}
	f.items[itemKey(service, account)] = secret
	return nil
}

func (f *fakeBackend) Get(service, account string) (string, error) {
	matches := strings.Contains(account, f.errAccount)
	if f.getBlock && matches {
		select {}
	}
	if f.getErr != nil && matches {
		return "", f.getErr
	}
	v, ok := f.items[itemKey(service, account)]
	if f.afterGet != nil {
		f.afterGet(account)
	}
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (f *fakeBackend) Delete(service, account string) error {
	matches := strings.Contains(account, f.errAccount)
	if f.deleteBlock && matches {
		select {}
	}
	if f.deleteErr != nil && matches {
		return f.deleteErr
	}
	k := itemKey(service, account)
	if _, ok := f.items[k]; !ok {
		return keyring.ErrNotFound
	}
	delete(f.items, k)
	return nil
}

// testTimeout bounds calls that fakeBackend answers at once. It is long
// because a loaded CI runner (seen on Windows) can take over 100ms to
// run the withTimeout goroutine, which failed tests with a spurious
// TimeoutError. Tests whose backend blocks forever use blockedTimeout
// so they still finish quickly.
const (
	testTimeout    = time.Minute
	blockedTimeout = 100 * time.Millisecond
)

func newTestStore(backend keyringBackend) *keyringStore {
	return newTestStoreWithTimeout(backend, testTimeout)
}

func newTestStoreWithTimeout(backend keyringBackend, timeout time.Duration) *keyringStore {
	return &keyringStore{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: "databricks-cli",
		// Tests make their own concurrent writes, so Lookup need not wait.
		retryDelay: time.Nanosecond,
	}
}

func TestKeyringStore_Store_WritesJSON(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	tok := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}

	require.NoError(t, c.Put("my-profile", Entry{Token: tok}))

	stored, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	require.True(t, ok, "token should be stored under service=databricks-cli, account=my-profile")

	var got keyringEntry
	require.NoError(t, json.Unmarshal([]byte(stored), &got))
	require.NotNil(t, got.Token)
	assert.Equal(t, "abc", got.Token.AccessToken)
	assert.Equal(t, "Bearer", got.Token.TokenType)
}

func TestKeyringStore_Store_PropagatesBackendError(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.setErr = boom
	c := newTestStore(backend)

	err := c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "x"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringStore_Lookup_ReturnsStoredToken(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	want := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}
	require.NoError(t, c.Put("my-profile", Entry{Token: want}))

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)
	assert.Equal(t, "Bearer", got.Token.TokenType)
}

func TestKeyringStore_Lookup_ReturnsStoredPendingLogin(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	want := &PendingLogin{DeviceCode: "device-code", UserCode: "WDJBMJHT"}
	require.NoError(t, c.Put("my-profile#device-code", Entry{PendingLogin: want}))

	got, err := c.Lookup("my-profile#device-code")
	require.NoError(t, err)
	assert.Nil(t, got.Token)
	assert.Equal(t, want, got.PendingLogin)
}

func TestKeyringStore_Lookup_MissingReturnsCacheErrNotFound(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	_, err := c.Lookup("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestKeyringStore_Lookup_PropagatesOtherErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.getErr = boom
	c := newTestStore(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	// The non-ErrNotFound path wraps the backend error with actionable
	// guidance so users on systems without a usable keyring backend
	// (e.g. headless Linux) know what to do.
	assert.Contains(t, err.Error(), "OS keyring unreachable")
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE=plaintext")
	assert.Contains(t, err.Error(), "databricks auth login")
}

// ErrNotFound has to pass through unwrapped because callers branch on it
// (ErrNotFound is the "no token, please log in" signal). Wrapping it
// with the unreachability hint would mislead the user.
func TestKeyringStore_Lookup_NotFoundIsNotWrapped(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	_, err := c.Lookup("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NotContains(t, err.Error(), "OS keyring unreachable")
}

func TestKeyringStore_Lookup_CorruptedJSONReturnsError(t *testing.T) {
	backend := newFakeBackend()
	backend.items[itemKey("databricks-cli", "my-profile")] = "{not json"
	c := newTestStore(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal token")
}

func TestKeyringStore_StoreNil_DeletesEntry(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	require.NoError(t, c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "abc"}}))
	require.NoError(t, c.Delete("my-profile"))

	_, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	assert.False(t, ok, "entry should be gone after delete")
}

func TestKeyringStore_StoreNil_MissingIsIdempotent(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteErr = keyring.ErrNotFound
	c := newTestStore(backend)

	err := c.Delete("never-stored")
	require.NoError(t, err, "deleting a missing entry must not error")
}

func TestKeyringStore_StoreNil_PropagatesOtherDeleteErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.deleteErr = boom
	c := newTestStore(backend)

	err := c.Delete("my-profile")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringStore_Store_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.setBlock = true
	c := newTestStoreWithTimeout(backend, blockedTimeout)

	start := time.Now()
	err := c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "x"}})
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
	assert.Less(t, time.Since(start), 2*time.Second, "should time out quickly")
}

func TestKeyringStore_Lookup_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.getBlock = true
	c := newTestStoreWithTimeout(backend, blockedTimeout)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}

func TestKeyringStore_StoreNil_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteBlock = true
	c := newTestStoreWithTimeout(backend, blockedTimeout)

	err := c.Delete("my-profile")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}

// testAccessToken returns an access token of n bytes in which no two parts
// are alike, so that a test notices parts in the wrong order.
func testAccessToken(n int) string {
	var b strings.Builder
	for i := 0; b.Len() < n; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('.')
	}
	return b.String()[:n]
}

// testToken returns a token with every field set and an access token of
// accessTokenSize bytes.
func testToken(accessTokenSize int) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  testAccessToken(accessTokenSize),
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC),
	}
}

// tooBigForAnyOS applies the macOS and the Windows cap together, because a
// token has to fit both for the store to behave the same everywhere.
func tooBigForAnyOS(service, account, secret string) bool {
	return darwinSecurityCommandTooBig(service, account, secret) || windowsCredentialBlobTooBig(service, account, secret)
}

// newLimitedTestStore returns a store on a fake backend that applies the
// macOS and the Windows cap.
func newLimitedTestStore() (*keyringStore, *fakeBackend) {
	backend := newFakeBackend()
	backend.secretTooBig = tooBigForAnyOS
	return newTestStore(backend), backend
}

// releasedEntry returns the item that released CLI versions write for tok and
// read back.
func releasedEntry(t *testing.T, tok *oauth2.Token) string {
	t.Helper()
	raw, err := json.Marshal(map[string]*oauth2.Token{"token": tok})
	require.NoError(t, err)
	return string(raw)
}

// itemsOf returns key followed by the keys of its first n access token parts.
func itemsOf(key string, n int) []string {
	items := []string{key}
	for i := range n {
		items = append(items, accessTokenPartKey(key, i))
	}
	return items
}

// maxTestParts is how many access token parts presentItems looks for, one
// more than the largest test token has.
const maxTestParts = 4

// presentItems returns which of key and its first maxTestParts access token
// parts the backend holds. It only calls Get, so it also inspects the real OS
// keyring, which cannot list its items.
func presentItems(t *testing.T, backend keyringBackend, key string) []string {
	t.Helper()
	var present []string
	for _, account := range itemsOf(key, maxTestParts) {
		_, err := backend.Get(keyringServiceName, account)
		if errors.Is(err, keyring.ErrNotFound) {
			continue
		}
		require.NoError(t, err)
		present = append(present, account)
	}
	return present
}

// largestAccessTokenInOneItem returns the size of the largest access token for
// which the released entry of testToken under key passes secretTooBig.
func largestAccessTokenInOneItem(t *testing.T, secretTooBig func(service, account, secret string) bool, key string) int {
	t.Helper()
	n := 0
	for !secretTooBig(keyringServiceName, key, releasedEntry(t, testToken(n+1))) {
		n++
	}
	return n
}

// Other CLI versions read the same entry, so a token that fits must keep the
// format they wrote and understand, under every OS cap.
func TestKeyringStore_Put_TokenThatFitsKeepsSingleItemFormat(t *testing.T) {
	for _, tc := range osSecretCaps {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.secretTooBig = tc.secretTooBig
			c := newTestStore(backend)
			tok := testToken(issue7707SmallAccessTokenBytes)

			require.NoError(t, c.Put("my-profile", Entry{Token: tok}))
			assert.Equal(t, map[string]string{itemKey(keyringServiceName, "my-profile"): releasedEntry(t, tok)}, backend.items)
		})
	}
}

// A token is split only when the backend rejects it as one item: the largest
// token that fits stays in the released format, and one byte more is split.
// The macOS cap counts the account too, so less fits under a long cache key
// such as a host URL.
func TestKeyringStore_Put_SplitsOnlyTokenThatDoesNotFit(t *testing.T) {
	keys := []struct {
		name string
		key  string
	}{
		{name: "profile key", key: "my-profile"},
		{name: "host key", key: "https://accounts.cloud.databricks.test/oidc/accounts/968367da-7edd-44f7-9dea-3e0b20b0ec97"},
	}

	for _, tc := range osSecretCaps {
		if tc.secretTooBig == nil {
			continue
		}
		for _, k := range keys {
			t.Run(tc.name+", "+k.name, func(t *testing.T) {
				n := largestAccessTokenInOneItem(t, tc.secretTooBig, k.key)
				backend := newFakeBackend()
				backend.secretTooBig = tc.secretTooBig
				c := newTestStore(backend)

				fits := testToken(n)
				require.NoError(t, c.Put(k.key, Entry{Token: fits}))
				assert.Equal(t, map[string]string{itemKey(keyringServiceName, k.key): releasedEntry(t, fits)}, backend.items)

				tooBig := testToken(n + 1)
				require.NoError(t, c.Put(k.key, Entry{Token: tooBig}))
				assert.Equal(t, itemsOf(k.key, 2), backend.accounts())
				got, err := c.Lookup(k.key)
				require.NoError(t, err)
				assert.Equal(t, tooBig.AccessToken, got.Token.AccessToken)
			})
		}
	}
}

func TestKeyringStore_Put_SplitsTokenTooBigForOneItem(t *testing.T) {
	cases := []struct {
		name            string
		accessTokenSize int
		wantParts       int
	}{
		{name: "over the Windows limit", accessTokenSize: 2600, wantParts: 2},
		// https://github.com/databricks/cli/issues/7707
		{name: "over the macOS limit", accessTokenSize: issue7707LargeAccessTokenBytes, wantParts: 2},
		{name: "three access token parts", accessTokenSize: 5000, wantParts: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, backend := newLimitedTestStore()
			want := testToken(tc.accessTokenSize)

			require.NoError(t, c.Put("my-profile", Entry{Token: want}))
			assert.Equal(t, itemsOf("my-profile", tc.wantParts), backend.accounts())

			got, err := c.Lookup("my-profile")
			require.NoError(t, err)
			assert.Equal(t, want.AccessToken, got.Token.AccessToken)
			assert.Equal(t, want.TokenType, got.Token.TokenType)
			assert.Equal(t, want.RefreshToken, got.Token.RefreshToken)
			assert.True(t, want.Expiry.Equal(got.Token.Expiry))
		})
	}
}

// CLI versions without split entries read the entry as {"token": ...}. They
// need a token there because releases before v1.24 dereference it, and it
// must not carry the refresh token: they would rotate it and then fail to
// store the oversized result, which logs the user out.
func TestKeyringStore_Put_SplitEntryGivesOlderVersionsNoCredentials(t *testing.T) {
	c, backend := newLimitedTestStore()

	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))

	var old struct {
		Token *oauth2.Token `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(backend.items[itemKey(keyringServiceName, "my-profile")]), &old))
	require.NotNil(t, old.Token)
	assert.Empty(t, old.Token.AccessToken)
	assert.Empty(t, old.Token.RefreshToken)
}

// Entry holds only a token and a pending login today, and the two never come
// together. The pending login stands in for fields that Entry may get later:
// a split token must keep them, as a token that fits does.
func TestKeyringStore_SplitEntryKeepsOtherEntryFields(t *testing.T) {
	c, _ := newLimitedTestStore()
	pending := &PendingLogin{DeviceCode: "device-code", UserCode: "WDJBMJHT"}
	tok := testToken(issue7707LargeAccessTokenBytes)

	require.NoError(t, c.Put("my-profile", Entry{Token: tok, PendingLogin: pending}))

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Equal(t, tok.AccessToken, got.Token.AccessToken)
	assert.Equal(t, pending, got.PendingLogin)
}

// The entry is written before the access token parts because it holds the
// refresh token, which the server replaces on every refresh. When a part write
// fails, whatever the key held before, the new refresh token is stored and
// read back without an access token.
func TestKeyringStore_Put_StoresRefreshTokenBeforeAccessToken(t *testing.T) {
	boom := errors.New("backend boom")
	before := []struct {
		name            string
		accessTokenSize int
	}{
		{name: "no entry"},
		{name: "token that fits", accessTokenSize: issue7707SmallAccessTokenBytes},
		{name: "split token", accessTokenSize: 5000},
	}

	for _, b := range before {
		for _, failingPart := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s, part %d fails", b.name, failingPart), func(t *testing.T) {
				c, backend := newLimitedTestStore()
				if b.accessTokenSize > 0 {
					old := testToken(b.accessTokenSize)
					old.RefreshToken = "old-refresh-token"
					require.NoError(t, c.Put("my-profile", Entry{Token: old}))
				}
				backend.setErr = boom
				backend.errAccount = accessTokenPartKey("my-profile", failingPart)
				tok := testToken(issue7707LargeAccessTokenBytes)

				err := c.Put("my-profile", Entry{Token: tok})
				require.ErrorIs(t, err, boom)
				assert.ErrorIs(t, err, ErrAccessTokenNotStored)

				got, err := c.Lookup("my-profile")
				require.NoError(t, err)
				assert.Equal(t, tok.RefreshToken, got.Token.RefreshToken)
				assert.Empty(t, got.Token.AccessToken)
			})
		}
	}
}

// Only the access token is split. Anything else that is too big for one item
// still fails, and nothing is written.
func TestKeyringStore_Put_SplitsOnlyTheAccessToken(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		entry Entry
	}{
		{
			name:  "pending login",
			key:   "my-profile#device-code",
			entry: Entry{PendingLogin: &PendingLogin{DeviceCode: strings.Repeat("x", 5000)}},
		},
		{
			name:  "refresh token",
			key:   "my-profile",
			entry: Entry{Token: &oauth2.Token{AccessToken: testAccessToken(issue7707LargeAccessTokenBytes), RefreshToken: strings.Repeat("x", 5000)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, backend := newLimitedTestStore()

			err := c.Put(tc.key, tc.entry)
			assert.ErrorIs(t, err, keyring.ErrSetDataTooBig)
			assert.NotErrorIs(t, err, ErrAccessTokenNotStored)
			assert.Empty(t, backend.items)
		})
	}
}

// editSplitEntry rewrites the split entry under my-profile.
func editSplitEntry(t *testing.T, backend *fakeBackend, edit func(*splitToken)) {
	t.Helper()
	k := itemKey(keyringServiceName, "my-profile")
	var e keyringEntry
	require.NoError(t, json.Unmarshal([]byte(backend.items[k]), &e))
	require.NotNil(t, e.Split)
	edit(e.Split)
	raw, err := json.Marshal(e)
	require.NoError(t, err)
	backend.items[k] = string(raw)
}

// The access token parts are written one at a time, so a reader can find them
// incomplete or belonging to another write. The refresh token is enough to
// recover, so the lookup succeeds without an access token.
func TestKeyringStore_Lookup_SplitEntryWithoutMatchingAccessToken(t *testing.T) {
	second := itemKey(keyringServiceName, "my-profile#access-token-1")
	cases := []struct {
		name   string
		change func(t *testing.T, backend *fakeBackend)
	}{
		{
			name:   "part missing",
			change: func(t *testing.T, backend *fakeBackend) { delete(backend.items, second) },
		},
		{
			name:   "part of another write",
			change: func(t *testing.T, backend *fakeBackend) { backend.items[second] = "another access token" },
		},
		{
			name: "no parts counted",
			change: func(t *testing.T, backend *fakeBackend) {
				editSplitEntry(t, backend, func(s *splitToken) { s.AccessTokenParts = 0 })
			},
		},
		{
			name: "fewer parts counted than written",
			change: func(t *testing.T, backend *fakeBackend) {
				editSplitEntry(t, backend, func(s *splitToken) { s.AccessTokenParts = 1 })
			},
		},
		{
			name: "more parts counted than written",
			change: func(t *testing.T, backend *fakeBackend) {
				editSplitEntry(t, backend, func(s *splitToken) { s.AccessTokenParts = 3 })
			},
		},
		{
			name: "digest of another access token",
			change: func(t *testing.T, backend *fakeBackend) {
				editSplitEntry(t, backend, func(s *splitToken) { s.AccessTokenSHA256 = strings.Repeat("0", 64) })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, backend := newLimitedTestStore()
			tok := testToken(issue7707LargeAccessTokenBytes)
			require.NoError(t, c.Put("my-profile", Entry{Token: tok}))
			require.Equal(t, itemsOf("my-profile", 2), backend.accounts())

			tc.change(t, backend)

			got, err := c.Lookup("my-profile")
			require.NoError(t, err)
			assert.Empty(t, got.Token.AccessToken)
			assert.Equal(t, tok.RefreshToken, got.Token.RefreshToken)
			assert.Equal(t, tok.TokenType, got.Token.TokenType)
			assert.True(t, tok.Expiry.Equal(got.Token.Expiry))
		})
	}
}

// splitItems returns the items that the store writes for tok under
// my-profile.
func splitItems(t *testing.T, tok *oauth2.Token) map[string]string {
	t.Helper()
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: tok}))
	return backend.items
}

// Another process can replace a split token while Lookup reads it. Here it
// writes its entry and first part after Lookup read the old entry, and its
// second part after Lookup read the old one. Lookup reads the entry again and
// returns the new token, rather than the old refresh token, which the other
// process has already used up.
func TestKeyringStore_Lookup_WaitsForConcurrentWrite(t *testing.T) {
	c, backend := newLimitedTestStore()
	old := testToken(issue7707LargeAccessTokenBytes)
	old.RefreshToken = "old-refresh-token"
	require.NoError(t, c.Put("my-profile", Entry{Token: old}))

	updated := testToken(issue7707LargeAccessTokenBytes)
	updated.AccessToken = strings.ReplaceAll(updated.AccessToken, ".", "-")
	updated.RefreshToken = "new-refresh-token"
	written := splitItems(t, updated)
	entry := itemKey(keyringServiceName, "my-profile")
	part0 := itemKey(keyringServiceName, "my-profile#access-token-0")
	part1 := itemKey(keyringServiceName, "my-profile#access-token-1")
	reads := 0
	backend.afterGet = func(string) {
		reads++
		switch reads {
		case 1:
			backend.items[entry] = written[entry]
			backend.items[part0] = written[part0]
		case 3:
			backend.items[part1] = written[part1]
		}
	}

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Equal(t, updated.AccessToken, got.Token.AccessToken)
	assert.Equal(t, updated.RefreshToken, got.Token.RefreshToken)
}

// auth logout can delete a split token while Lookup reads it. Lookup reads the
// entry again and finds it gone, rather than returning the deleted refresh
// token, which the caller would refresh and store again.
func TestKeyringStore_Lookup_SplitEntryDeletedDuringLookup(t *testing.T) {
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))
	backend.afterGet = func(account string) {
		if account == "my-profile#access-token-0" {
			clear(backend.items)
		}
	}

	_, err := c.Lookup("my-profile")
	assert.ErrorIs(t, err, ErrNotFound)
}

// A split entry whose parts stay incomplete, as after an interrupted write, is
// read splitLookupAttempts times and then returned without its access token.
func TestKeyringStore_Lookup_GivesUpOnIncompleteSplitEntry(t *testing.T) {
	c, backend := newLimitedTestStore()
	tok := testToken(issue7707LargeAccessTokenBytes)
	require.NoError(t, c.Put("my-profile", Entry{Token: tok}))
	delete(backend.items, itemKey(keyringServiceName, "my-profile#access-token-1"))
	entryReads := 0
	backend.afterGet = func(account string) {
		if account == "my-profile" {
			entryReads++
		}
	}

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Empty(t, got.Token.AccessToken)
	assert.Equal(t, tok.RefreshToken, got.Token.RefreshToken)
	assert.Equal(t, splitLookupAttempts, entryReads)
}

// A store without a retry delay, as NewKeyringStore builds it, waits 25, 50,
// 100 and 200 milliseconds between its reads of an incomplete split entry, to
// give a concurrent writer time to finish.
func TestKeyringStore_Lookup_WaitsBetweenReadsByDefault(t *testing.T) {
	backend := newFakeBackend()
	backend.secretTooBig = tooBigForAnyOS
	c := &keyringStore{backend: backend, timeout: testTimeout, keyringSvcName: keyringServiceName}
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))
	delete(backend.items, itemKey(keyringServiceName, "my-profile#access-token-1"))

	start := time.Now()
	_, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 375*time.Millisecond)
}

func TestKeyringStore_Lookup_SplitEntryPropagatesAccessTokenErrors(t *testing.T) {
	boom := errors.New("backend boom")
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))
	backend.getErr = boom
	backend.errAccount = AccessTokenPartMarker

	_, err := c.Lookup("my-profile")
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "OS keyring unreachable")
}

func TestKeyringStore_Delete_PropagatesAccessTokenPartErrors(t *testing.T) {
	boom := errors.New("backend boom")
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))
	backend.deleteErr = boom
	backend.errAccount = AccessTokenPartMarker

	err := c.Delete("my-profile")
	assert.ErrorIs(t, err, boom)
}

// Every access token part is read, written and deleted under the timeout of
// a single keyring operation.
func TestKeyringStore_AccessTokenPartTimesOut(t *testing.T) {
	cases := []struct {
		name  string
		block func(*fakeBackend)
		run   func(*keyringStore) error
	}{
		{
			name:  "lookup",
			block: func(b *fakeBackend) { b.getBlock = true },
			run: func(c *keyringStore) error {
				_, err := c.Lookup("my-profile")
				return err
			},
		},
		{
			name:  "put",
			block: func(b *fakeBackend) { b.setBlock = true },
			run: func(c *keyringStore) error {
				return c.Put("my-profile", Entry{Token: testToken(5000)})
			},
		},
		{
			name:  "delete",
			block: func(b *fakeBackend) { b.deleteBlock = true },
			run: func(c *keyringStore) error {
				return c.Delete("my-profile")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, backend := newLimitedTestStore()
			require.NoError(t, c.Put("my-profile", Entry{Token: testToken(issue7707LargeAccessTokenBytes)}))
			backend.errAccount = AccessTokenPartMarker
			tc.block(backend)
			c.timeout = blockedTimeout

			err := tc.run(c)

			var timeoutErr *TimeoutError
			assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
		})
	}
}

// assertTokenSizeTransitions stores tokens of changing sizes under key, checks
// that each one reads back and which items exist, and then deletes key.
func assertTokenSizeTransitions(t *testing.T, c *keyringStore, backend keyringBackend, key string) {
	t.Helper()
	steps := []struct {
		name            string
		accessTokenSize int
		wantParts       int
	}{
		{name: "token that fits", accessTokenSize: issue7707SmallAccessTokenBytes, wantParts: 0},
		{name: "split into three parts", accessTokenSize: 5000, wantParts: 3},
		// The third part stays behind, unused: the entry counts two.
		{name: "split into two parts", accessTokenSize: issue7707LargeAccessTokenBytes, wantParts: 3},
		// All parts stay behind until Delete.
		{name: "token that fits again", accessTokenSize: issue7707SmallAccessTokenBytes, wantParts: 3},
	}

	for i, s := range steps {
		tok := testToken(s.accessTokenSize)
		tok.RefreshToken = fmt.Sprintf("refresh-token-%d", i)
		require.NoError(t, c.Put(key, Entry{Token: tok}), s.name)

		got, err := c.Lookup(key)
		require.NoError(t, err, s.name)
		assert.Equal(t, tok.AccessToken, got.Token.AccessToken, s.name)
		assert.Equal(t, tok.RefreshToken, got.Token.RefreshToken, s.name)
		assert.Equal(t, itemsOf(key, s.wantParts), presentItems(t, backend, key), s.name)
	}

	require.NoError(t, c.Delete(key))
	assert.Empty(t, presentItems(t, backend, key))
}

// The access token's size changes between refreshes, so one key moves between
// a token that fits and a split one. Delete also removes the parts that the
// entry no longer counts.
func TestKeyringStore_TokenSizeTransitions(t *testing.T) {
	c, backend := newLimitedTestStore()
	assertTokenSizeTransitions(t, c, backend, "my-profile")
	assert.Empty(t, backend.items)
}

// An older CLI version that logs out deletes only the entry, so Delete also
// removes access token parts without one.
func TestKeyringStore_Delete_RemovesPartsWithoutEntry(t *testing.T) {
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(5000)}))
	delete(backend.items, itemKey(keyringServiceName, "my-profile"))

	require.NoError(t, c.Delete("my-profile"))
	assert.Empty(t, backend.items)
}

func TestKeyringStore_Delete_MissingEntryIsNoOp(t *testing.T) {
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("other-profile", Entry{Token: testToken(5000)}))

	require.NoError(t, c.Delete("my-profile"))
	assert.Equal(t, itemsOf("other-profile", 3), backend.accounts())
}

// A Delete that fails part way, here on the second of three parts, leaves
// parts that the next Delete still finds, so a retried logout removes them.
func TestKeyringStore_Delete_RetryAfterFailedPartDelete(t *testing.T) {
	boom := errors.New("backend boom")
	c, backend := newLimitedTestStore()
	require.NoError(t, c.Put("my-profile", Entry{Token: testToken(5000)}))
	backend.deleteErr = boom
	backend.errAccount = accessTokenPartKey("my-profile", 1)
	require.ErrorIs(t, c.Delete("my-profile"), boom)

	backend.deleteErr = nil
	require.NoError(t, c.Delete("my-profile"))
	assert.Empty(t, backend.items)
}

func TestProbeKeyring(t *testing.T) {
	boom := errors.New("backend boom")
	cases := []struct {
		name        string
		setErr      error
		deleteErr   error
		setBlock    bool
		timeout     time.Duration
		wantErr     error
		wantTimeout bool
	}{
		{
			name:    "success leaves no entry",
			timeout: testTimeout,
		},
		{
			name:    "set error propagates",
			setErr:  boom,
			timeout: testTimeout,
			wantErr: boom,
		},
		{
			name:        "set times out",
			setBlock:    true,
			timeout:     blockedTimeout,
			wantTimeout: true,
		},
		{
			name:      "delete error propagates",
			deleteErr: boom,
			timeout:   testTimeout,
			wantErr:   boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.setErr = tc.setErr
			backend.deleteErr = tc.deleteErr
			backend.setBlock = tc.setBlock

			err := probeWithBackend(backend, tc.timeout)

			switch {
			case tc.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			case tc.wantTimeout:
				require.Error(t, err)
				var timeoutErr *TimeoutError
				assert.ErrorAs(t, err, &timeoutErr)
			default:
				require.NoError(t, err)
				assert.Empty(t, backend.items, "probe must clean up after itself")
			}
		})
	}
}

func TestProbeKeyringRead(t *testing.T) {
	boom := errors.New("backend boom")
	cases := []struct {
		name        string
		getErr      error
		getBlock    bool
		timeout     time.Duration
		wantErr     error
		wantTimeout bool
	}{
		{
			// keyring.ErrNotFound is the success signal: the backend
			// responded that no entry exists for our probe account,
			// which means it is reachable.
			name:    "ErrNotFound counts as reachable",
			getErr:  keyring.ErrNotFound,
			timeout: testTimeout,
		},
		{
			name:    "other backend error propagates",
			getErr:  boom,
			timeout: testTimeout,
			wantErr: boom,
		},
		{
			name:        "get times out",
			getBlock:    true,
			timeout:     blockedTimeout,
			wantTimeout: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.getErr = tc.getErr
			backend.getBlock = tc.getBlock

			err := probeReadWithBackend(backend, tc.timeout)

			switch {
			case tc.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			case tc.wantTimeout:
				require.Error(t, err)
				var timeoutErr *TimeoutError
				assert.ErrorAs(t, err, &timeoutErr)
			default:
				require.NoError(t, err)
				assert.Empty(t, backend.items, "read probe must not write to the keyring")
			}
		})
	}
}

package u2m_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/databricks-sdk-go/httpclient/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const fileRefreshActor = "DATABRICKS_TEST_FILE_REFRESH_ACTOR"

type actorMessage struct {
	Kind       string
	Generation int
	Failed     bool
	Redacted   bool
}

type actorSetup struct {
	Path  string
	Force bool
}

type gatedFileStore struct {
	storage.Store
	input  *json.Decoder
	output *json.Encoder
	first  atomic.Bool
}

func (s *gatedFileStore) Lookup(key string) (storage.Entry, error) {
	entry, err := s.Store.Lookup(key)
	if err != nil || s.first.Swap(true) {
		return entry, err
	}
	if err := s.output.Encode(actorMessage{Kind: "initial-read", Generation: tokenGeneration(entry.Token.AccessToken)}); err != nil {
		return storage.Entry{}, err
	}
	var release actorMessage
	if err := s.input.Decode(&release); err != nil {
		return storage.Entry{}, err
	}
	if release.Kind != "release" {
		return storage.Entry{}, errors.New("unexpected initial-read control")
	}
	return entry, nil
}

// The structural interface keeps this regression usable on the older owner too.
func (s *gatedFileStore) WithFileTransaction(ctx context.Context, update, cleanup func(storage.Store) error) (bool, error) {
	owner, ok := s.Store.(interface {
		WithFileTransaction(context.Context, func(storage.Store) error, func(storage.Store) error) (bool, error)
	})
	if !ok {
		return false, nil
	}
	return owner.WithFileTransaction(ctx, update, cleanup)
}

type fileRefreshTransport struct {
	input  *json.Decoder
	output *json.Encoder
}

func (r fileRefreshTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	form, err := url.ParseQuery(string(body))
	if err != nil || form.Get("grant_type") != "refresh_token" {
		return nil, errors.New("unexpected synthetic exchange")
	}
	if err := r.output.Encode(actorMessage{Kind: "exchange", Generation: tokenGeneration(form.Get("refresh_token"))}); err != nil {
		return nil, err
	}
	var reply actorMessage
	if err := r.input.Decode(&reply); err != nil {
		return nil, err
	}
	if reply.Kind != "exchange-result" {
		return nil, errors.New("unexpected exchange control")
	}
	status := http.StatusOK
	response := fmt.Sprintf(`{"access_token":"generation-%d","refresh_token":"generation-%d","token_type":"Bearer","expires_in":3600}`, reply.Generation, reply.Generation)
	if reply.Failed {
		status = http.StatusBadRequest
		response = `{"error":"invalid_grant","error_description":"Refresh token is invalid"}`
	}
	transport := fixtures.SliceTransport{{
		Method: "POST", Resource: "/oidc/accounts/account/v1/token",
		Status: status, Response: response,
		ResponseHeaders: map[string][]string{"Content-Type": {"application/json"}},
	}}
	return transport.RoundTrip(request)
}

func tokenGeneration(token string) int {
	generation, err := strconv.Atoi(strings.TrimPrefix(token, "generation-"))
	if err != nil {
		return -1
	}
	return generation
}

func runFileRefreshActor(t *testing.T) {
	t.Helper()
	input := json.NewDecoder(os.Stdin)
	output := json.NewEncoder(os.Stdout)
	var setup actorSetup
	require.NoError(t, input.Decode(&setup), "actor setup")
	store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(setup.Path))
	require.NoError(t, err, "actor file store")
	arg, err := u2m.NewBasicAccountOAuthArgument("https://accounts.databricks.test", "account")
	require.NoError(t, err)
	provider, err := u2m.NewPersistentAuth(t.Context(),
		u2m.WithTokenStore(&gatedFileStore{Store: store, input: input, output: output}),
		u2m.WithOAuthArgument(arg), u2m.WithClientID("synthetic-client"),
		u2m.WithOAuthEndpointSupplier(&u2m.BasicOAuthEndpointSupplier{}),
		u2m.WithHttpClient(&http.Client{Transport: fileRefreshTransport{input: input, output: output}}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })
	var token *oauth2.Token
	if setup.Force {
		token, err = provider.ForceRefreshToken()
	} else {
		token, err = provider.Token()
	}
	result := actorMessage{Kind: "result", Failed: err != nil}
	if token != nil {
		result.Generation = tokenGeneration(token.AccessToken)
		result.Redacted = token.RefreshToken == ""
	}
	require.NoError(t, output.Encode(result))
}

type refreshChild struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  *json.Decoder
	waited  bool
}

func startFileRefreshActor(t *testing.T, home string, setup actorSetup) *refreshChild {
	t.Helper()
	require.NoError(t, os.MkdirAll(home, 0o700))
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFileStoreRefreshAcrossProcesses$", "-test.timeout=30s")
	command.Env = []string{
		fileRefreshActor + "=1", "HOME=" + home, "USERPROFILE=" + home,
		"PATH=" + os.Getenv("PATH"), "TMPDIR=" + os.TempDir(), "TMP=" + os.TempDir(), "TEMP=" + os.TempDir(),
	}
	input, err := command.StdinPipe()
	require.NoError(t, err)
	output, err := command.StdoutPipe()
	require.NoError(t, err)
	command.Stderr = &bytes.Buffer{}
	require.NoError(t, command.Start())
	child := &refreshChild{command: command, input: input, output: json.NewDecoder(output)}
	t.Cleanup(func() {
		_ = input.Close()
		if !child.waited {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	require.NoError(t, json.NewEncoder(input).Encode(setup))
	return child
}

func TestFileStoreRefreshAcrossProcesses(t *testing.T) {
	if os.Getenv(fileRefreshActor) == "1" {
		runFileRefreshActor(t)
		return
	}
	for _, test := range []struct {
		name      string
		sameHome  bool
		force     bool
		expired   bool
		exchanges int
	}{
		{name: "shared-path-different-homes", exchanges: 1},
		{name: "near-expiry-same-home", sameHome: true, exchanges: 1},
		{name: "expired-shared-path", expired: true, exchanges: 1},
		{name: "explicit-force-same-home", sameHome: true, force: true, exchanges: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "shared", "tokens.json")
			store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
			require.NoError(t, err)
			arg, err := u2m.NewBasicAccountOAuthArgument("https://accounts.databricks.test", "account")
			require.NoError(t, err)
			expiry := time.Now().Add(time.Minute)
			if test.expired {
				expiry = time.Now().Add(-time.Minute)
			} else if test.force {
				expiry = time.Now().Add(time.Hour)
			}
			require.NoError(t, store.Put(arg.GetCacheKey(), storage.Entry{Token: &oauth2.Token{
				AccessToken: "generation-0", RefreshToken: "generation-0", Expiry: expiry,
			}}))
			require.NoError(t, store.Put("unrelated", storage.Entry{Token: &oauth2.Token{AccessToken: "unrelated"}}))
			children := make([]*refreshChild, 2)
			for actor := range children {
				home := filepath.Join(root, fmt.Sprintf("home-%d", actor))
				if test.sameHome {
					home = filepath.Join(root, "home-shared")
				}
				children[actor] = startFileRefreshActor(t, home, actorSetup{Path: path, Force: test.force})
			}
			for _, child := range children {
				var ready actorMessage
				require.NoError(t, child.output.Decode(&ready))
				require.Equal(t, "initial-read", ready.Kind)
				require.Zero(t, ready.Generation)
			}
			type event struct {
				actor int
				value actorMessage
			}
			events := make(chan event, 4)
			for actor, child := range children {
				require.NoError(t, json.NewEncoder(child.input).Encode(actorMessage{Kind: "release"}))
				go func() {
					for {
						var value actorMessage
						if err := child.output.Decode(&value); err != nil {
							events <- event{actor: actor, value: actorMessage{Kind: "stream-error"}}
							return
						}
						events <- event{actor: actor, value: value}
						if value.Kind == "result" {
							return
						}
					}
				}()
			}
			generation, exchanges, replays := 0, 0, 0
			results := make([]actorMessage, 0, 2)
			for len(results) < 2 {
				event := <-events
				switch event.value.Kind {
				case "exchange":
					exchanges++
					replay := event.value.Generation != generation
					if replay {
						replays++
					} else {
						generation++
					}
					require.NoError(t, json.NewEncoder(children[event.actor].input).Encode(actorMessage{Kind: "exchange-result", Generation: generation, Failed: replay}))
				case "result":
					results = append(results, event.value)
				default:
					t.Fatal("unexpected actor stage")
				}
			}
			for _, child := range children {
				require.NoError(t, child.command.Wait(), "actor completion")
				child.waited = true
			}
			assert.Equal(t, test.exchanges, exchanges)
			assert.Zero(t, replays)
			returned := make([]int, 0, 2)
			for _, result := range results {
				assert.False(t, result.Failed)
				assert.True(t, result.Redacted)
				returned = append(returned, result.Generation)
			}
			slices.Sort(returned)
			if test.force {
				assert.Equal(t, []int{1, 2}, returned)
			} else {
				assert.Equal(t, []int{1, 1}, returned)
			}
			persisted, err := store.Lookup(arg.GetCacheKey())
			require.NoError(t, err)
			assert.Equal(t, test.exchanges, tokenGeneration(persisted.Token.AccessToken))
			assert.Equal(t, test.exchanges, tokenGeneration(persisted.Token.RefreshToken))
			other, err := store.Lookup("unrelated")
			require.NoError(t, err)
			assert.Equal(t, "unrelated", other.Token.AccessToken)
		})
	}
}

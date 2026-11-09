package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func intentFixture(t *testing.T) *fileStore {
	t.Helper()
	store, err := NewFileStore(t.Context(), WithFileLocation(filepath.Join(t.TempDir(), "tokens.json")))
	if err != nil {
		t.Fatal("file fixture setup failed")
	}
	return store.(*fileStore)
}

func intentToken(g string) Entry {
	return Entry{Token: &oauth2.Token{AccessToken: "access-" + g, RefreshToken: "refresh-" + g, Expiry: time.Now().Add(time.Hour)}}
}

func intentRefresh(t *testing.T, c *fileStore, key string, result Entry, calls *int) error {
	t.Helper()
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		a, err := BeginFileRefresh(t.Context(), view, key, sha256.Sum256([]byte("synthetic-route")))
		if err != nil {
			return err
		}
		*calls++
		if err := a.RecordResponse(result); err != nil {
			return err
		}
		return view.Put(key, result)
	}, nil)
	return err
}

func seedIntent(t *testing.T, c *fileStore, key string, e Entry) {
	t.Helper()
	if err := c.Put(key, e); err != nil {
		t.Fatal("seed failed")
	}
}

func readIntent(t *testing.T, c *fileStore) *refreshJournal {
	t.Helper()
	j, err := c.loadRefreshJournal(t.Context())
	if err != nil || j == nil {
		t.Fatal("intent load failed")
	}
	return j
}

func TestRefreshIntentPrepareFailureHasNoExchange(t *testing.T) {
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	before, _ := os.ReadFile(c.fileLocation)
	calls := 0
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		if err := os.Mkdir(c.fileLocation+refreshIntentSuffix, 0o700); err != nil {
			t.Fatal("fault setup failed")
		}
		_, err := BeginFileRefresh(t.Context(), view, "primary", sha256.Sum256([]byte("synthetic-route")))
		if err != nil {
			return err
		}
		calls++
		return nil
	}, nil)
	after, _ := os.ReadFile(c.fileLocation)
	if !errors.Is(err, ErrRefreshIntentIO) || calls != 0 || !bytes.Equal(before, after) {
		t.Fatal("prepare failure permitted exchange or cache mutation")
	}
}

func TestRefreshIntentAliasesAndIndependentGeneration(t *testing.T) {
	c := intentFixture(t)
	old := intentToken("0")
	for _, key := range []string{"profile", "legacy", "copy"} {
		seedIntent(t, c, key, old)
	}
	seedIntent(t, c, "other", intentToken("9"))
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		_, err := BeginFileRefresh(t.Context(), view, "profile", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		return errors.New("closed uncertain exchange")
	}, nil)
	if err == nil {
		t.Fatal("uncertain fixture did not fail")
	}
	j := readIntent(t, c)
	g := j.generation(old)
	prior := *j.Generations[g]
	calls := 0
	for _, key := range []string{"profile", "legacy", "copy"} {
		if !errors.Is(intentRefresh(t, c, key, intentToken("1"), &calls), ErrRefreshRecoveryRequired) {
			t.Fatal("alias reopened consumed credential")
		}
	}
	if err := intentRefresh(t, c, "other", intentToken("10"), &calls); err != nil {
		t.Fatal("independent generation blocked")
	}
	current := readIntent(t, c)
	if calls != 1 || !reflect.DeepEqual(prior, *current.Generations[g]) {
		t.Fatal("independent refresh changed consumed intent")
	}
	// Generic access-only and pending writes cannot open an epoch.
	seedIntent(t, c, "profile", Entry{Token: &oauth2.Token{AccessToken: "access-only"}})
	seedIntent(t, c, "pending", Entry{PendingLogin: &PendingLogin{DeviceCode: "synthetic", Host: "synthetic"}})
	seedIntent(t, c, "copy", old)
	if !errors.Is(intentRefresh(t, c, "copy", intentToken("1"), &calls), ErrRefreshRecoveryRequired) || calls != 1 {
		t.Fatal("generic writer reopened consumed credential")
	}
}

type intentPause struct{}

func captureLoginReady(t *testing.T) (*fileStore, *refreshJournal, string) {
	t.Helper()
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	calls := 0
	if err := intentRefresh(t, c, "primary", intentToken("1"), &calls); err != nil {
		t.Fatal("initial completed generation failed")
	}
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		_, err := BeginFileRefresh(t.Context(), view, "primary", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		return errors.New("closed stop")
	}, nil)
	if err == nil {
		t.Fatal("pending fixture failed")
	}
	c.beforeIntentCompletion = func() { panic(intentPause{}) }
	func() {
		defer func() {
			if _, ok := recover().(intentPause); !ok {
				t.Fatal("response publication boundary was not reached")
			}
		}()
		_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
			entry := Entry{Token: &oauth2.Token{AccessToken: "access-only"}}
			if err := RecordFileLoginResult(view, "primary", entry); err != nil {
				return err
			}
			return view.Put("primary", entry)
		}, nil)
		if err != nil {
			t.Fatal("login response proof failed")
		}
	}()
	c.beforeIntentCompletion = nil
	j := readIntent(t, c)
	for nonce, a := range j.Attempts {
		if a.Kind == "login" && a.Phase == "response_ready" {
			return c, j, nonce
		}
	}
	t.Fatal("response proof missing")
	return nil, nil, ""
}

func TestRefreshIntentNonceAndReplacementOwnership(t *testing.T) {
	_, j, nonce := captureLoginReady(t)
	a := j.Attempts[nonce]
	if !j.canComplete(nonce, a) || !validAttempt(nonce, a) {
		t.Fatal("captured owner cannot complete")
	}
	future, _ := intentNonce()
	if validAttempt(future, a) {
		t.Fatal("access-only proof was transferable to another nonce")
	}
	for _, variant := range []string{"reservation", "binding-entry", "owning-entry"} {
		t.Run(variant, func(t *testing.T) {
			copy, err := cloneRefreshJournal(j)
			if err != nil {
				t.Fatal("clone failed")
			}
			candidate := copy.Attempts[nonce]
			w := candidate.Writes[0]
			g := deref(w.BeforeGeneration)
			switch variant {
			case "reservation":
				old := copy.Attempts[deref(w.BeforeReservation)]
				next := *old
				copy.Attempts[future] = &next
				copy.Generations[g].Reservation = nullable(future)
			case "binding-entry":
				copy.Bindings[w.Key].Entry = intentHash("different-entry")
			case "owning-entry":
				copy.Attempts[deref(w.BeforeReservation)].InputEntry = nullable(intentHash("different-entry"))
			}
			if copy.canComplete(nonce, candidate) {
				t.Fatal("completion borrowed future or changed ownership")
			}
		})
	}
}

func TestRefreshIntentInvalidEntryAndCapabilityLifetime(t *testing.T) {
	if (&FileRefreshStateError{}).Error() == "" {
		t.Fatal("zero error has no safe message")
	}
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	var saved *FileRefreshAttempt
	calls := 0
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		a, err := BeginFileRefresh(t.Context(), view, "primary", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		saved = a
		calls++
		bad := intentToken("1")
		bad.Token.Expiry = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		return a.RecordResponse(bad)
	}, nil)
	if !errors.Is(err, ErrRefreshIntentInvalid) || calls != 1 || saved.response != nil {
		t.Fatal("invalid entry certified a response")
	}
	if saved.RecordResponse(intentToken("1")) == nil {
		t.Fatal("expired capability accepted response")
	}
	if !errors.Is(intentRefresh(t, c, "primary", intentToken("1"), &calls), ErrRefreshRecoveryRequired) || calls != 1 {
		t.Fatal("encoding failure reopened consumed generation")
	}
}

func TestRefreshIntentWholeFileAndTransitionValidation(t *testing.T) {
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	calls := 0
	if err := intentRefresh(t, c, "primary", intentToken("1"), &calls); err != nil {
		t.Fatal("completed setup failed")
	}
	raw, err := os.ReadFile(c.fileLocation + refreshIntentSuffix)
	if err != nil {
		t.Fatal("read failed")
	}
	for _, variant := range []string{"partial-length", "partial-frame", "checksum", "trailing"} {
		t.Run(variant, func(t *testing.T) {
			changed := append([]byte{}, raw...)
			switch variant {
			case "partial-length":
				changed = changed[:9]
			case "partial-frame":
				changed = changed[:len(changed)-1]
			case "checksum":
				changed[len(changed)-1] ^= 1
			case "trailing":
				changed = append(changed, 0)
			}
			if loaded, err := decodeRefreshJournal(changed, c.fileLocation); loaded != nil || !errors.Is(err, ErrRefreshIntentInvalid) {
				t.Fatal("invalid tail authorized a valid prefix")
			}
		})
	}
	j := readIntent(t, c)
	next, err := cloneRefreshJournal(j)
	if err != nil {
		t.Fatal("clone failed")
	}
	for _, g := range next.Generations {
		if g.Consumed {
			g.Consumed = false
			break
		}
	}
	if _, err := makeRefreshDelta(j, next, j.sequence+1); !errors.Is(err, ErrRefreshIntentInvalid) {
		t.Fatal("checksum-valid semantic rollback was permitted")
	}
	file, err := os.OpenFile(c.fileLocation+refreshIntentSuffix, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal("tail fault open failed")
	}
	_, _ = file.Seek(0, 2)
	_, err = file.Write([]byte{0})
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		t.Fatal("tail fault failed")
	}
	_, err = WithFileTransaction(t.Context(), c, func(view Store) error { calls++; return nil }, nil)
	if !errors.Is(err, ErrRefreshIntentInvalid) || calls != 1 {
		t.Fatal("partial file did not block entire transaction")
	}
}

func TestRefreshIntentOriginalContextAndPersistenceOrdering(t *testing.T) {
	for _, afterPrepare := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel-before-prepare", true: "cancel-after-prepare"}[afterPrepare], func(t *testing.T) {
			c := intentFixture(t)
			seedIntent(t, c, "primary", intentToken("0"))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			_, err := WithFileTransaction(ctx, c, func(view Store) error {
				if !afterPrepare {
					cancel()
				}
				_, err := BeginFileRefresh(t.Context(), view, "primary", sha256.Sum256([]byte("route")))
				if err != nil {
					return err
				}
				cancel()
				return ctx.Err()
			}, nil)
			if !errors.Is(err, context.Canceled) || calls != 0 {
				t.Fatal("original context was reset")
			}
			if afterPrepare && !errors.Is(intentRefresh(t, c, "primary", intentToken("1"), &calls), ErrRefreshRecoveryRequired) {
				t.Fatal("cancellation reopened prepared generation")
			}
		})
	}
	for _, afterCache := range []bool{false, true} {
		t.Run(map[bool]string{false: "response-proof-IO", true: "completion-IO"}[afterCache], func(t *testing.T) {
			c := intentFixture(t)
			seedIntent(t, c, "primary", intentToken("0"))
			path := c.fileLocation + refreshIntentSuffix
			calls := 0
			t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
			if afterCache {
				c.beforeIntentCompletion = func() {
					if err := os.Chmod(path, 0o400); err != nil {
						t.Fatal("fault setup failed")
					}
				}
			}
			_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
				a, err := BeginFileRefresh(t.Context(), view, "primary", sha256.Sum256([]byte("route")))
				if err != nil {
					return err
				}
				calls++
				result := intentToken("1")
				if err := a.RecordResponse(result); err != nil {
					return err
				}
				if err := view.Put("primary", result); err != nil {
					return err
				}
				if !afterCache {
					return os.Chmod(path, 0o400)
				}
				return nil
			}, nil)
			c.beforeIntentCompletion = nil
			if !errors.Is(err, ErrRefreshIntentIO) || calls != 1 {
				t.Fatal("persistence failure incorrectly succeeded")
			}
			if err := os.Chmod(path, 0o600); err != nil {
				t.Fatal("storage repair failed")
			}
			if afterCache {
				_, err = WithFileTransaction(t.Context(), c, func(view Store) error { return nil }, nil)
				if err != nil {
					t.Fatal("completed cache proof did not reconcile")
				}
			} else if !errors.Is(intentRefresh(t, c, "primary", intentToken("1"), &calls), ErrRefreshRecoveryRequired) || calls != 1 {
				t.Fatal("lost response proof permitted replay")
			}
		})
	}
}

func TestRefreshIntentRetirementOwnerAndSameCredentialLogin(t *testing.T) {
	for _, ownerDelete := range []bool{true, false} {
		t.Run(map[bool]string{true: "owner", false: "copied-alias"}[ownerDelete], func(t *testing.T) {
			c := intentFixture(t)
			original := intentToken("0")
			seedIntent(t, c, "owner", original)
			seedIntent(t, c, "copy", original)
			_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
				_, err := BeginFileRefresh(t.Context(), view, "owner", sha256.Sum256([]byte("route")))
				if err != nil {
					return err
				}
				return errors.New("closed unknown exchange")
			}, nil)
			if err == nil {
				t.Fatal("prepared fixture failed")
			}
			prior := readIntent(t, c)
			g := prior.generation(original)
			reservation := deref(prior.Generations[g].Reservation)
			key := "copy"
			if ownerDelete {
				key = "owner"
			}
			if err := c.Delete(key); err != nil {
				t.Fatal("captured deletion failed")
			}
			current := readIntent(t, c)
			if !current.Generations[g].Consumed || current.Generations[g].ActiveEpoch != nil {
				t.Fatal("Delete reopened consumed generation")
			}
			if ownerDelete {
				if current.Generations[g].Reservation != nil {
					t.Fatal("owner deletion retained its captured reservation")
				}
				calls := 0
				if !errors.Is(intentRefresh(t, c, "copy", intentToken("1"), &calls), ErrRefreshRecoveryRequired) || calls != 0 {
					t.Fatal("Delete authorized an alias")
				}
				result := intentToken("0")
				_, err = WithFileTransaction(t.Context(), c, func(view Store) error {
					if err := RecordFileLoginResult(view, "owner", result); err != nil {
						return err
					}
					return view.Put("owner", result)
				}, nil)
				if err != nil {
					t.Fatal("actual fresh login could not open same opaque credential")
				}
				if !errors.Is(intentRefresh(t, c, "copy", intentToken("1"), &calls), ErrRefreshRecoveryRequired) || calls != 0 {
					t.Fatal("fresh login opened an unrelated copied key")
				}
				if err := intentRefresh(t, c, "owner", intentToken("1"), &calls); err != nil || calls != 1 {
					t.Fatal("fresh login did not open only its result epoch")
				}
			} else {
				if deref(current.Generations[g].Reservation) != reservation {
					t.Fatal("deleting copied alias cleared original owner")
				}
				result := intentToken("0")
				_, err = WithFileTransaction(t.Context(), c, func(view Store) error {
					if err := RecordFileLoginResult(view, "copy", result); err != nil {
						return err
					}
					return view.Put("copy", result)
				}, nil)
				if !errors.Is(err, ErrRefreshRecoveryRequired) {
					t.Fatal("alias login borrowed original owner's reservation")
				}
			}
		})
	}
}

func TestRefreshIntentRetirementRejectsStaleCapture(t *testing.T) {
	c := intentFixture(t)
	entry := intentToken("0")
	seedIntent(t, c, "owner", entry)
	_, _ = WithFileTransaction(t.Context(), c, func(view Store) error {
		_, err := BeginFileRefresh(t.Context(), view, "owner", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		return errors.New("closed stop")
	}, nil)
	j := readIntent(t, c)
	g := j.generation(entry)
	reservation := deref(j.Generations[g].Reservation)
	owner := j.Attempts[reservation]
	proof := refreshRetirement{Key: owner.InputKey, Generation: g, Entry: deref(owner.InputEntry), Reservation: nullable(reservation)}
	for _, variant := range []string{"future-reservation", "changed-entry"} {
		t.Run(variant, func(t *testing.T) {
			copy, err := cloneRefreshJournal(j)
			if err != nil {
				t.Fatal("clone failed")
			}
			captured := proof
			if variant == "future-reservation" {
				future, _ := intentNonce()
				next := *copy.Attempts[reservation]
				copy.Attempts[future] = &next
				copy.Generations[g].Reservation = nullable(future)
			} else {
				captured.Entry = intentHash("changed-entry")
			}
			before, err := cloneRefreshJournal(copy)
			if err != nil {
				t.Fatal("clone failed")
			}
			if _, err := copy.retireCaptured(captured); !errors.Is(err, ErrRefreshRecoveryRequired) || !sameRefreshState(copy, before) {
				t.Fatal("stale retirement changed a future owner")
			}
		})
	}
}

type intentAliasKeys struct{}

func (intentAliasKeys) GetCacheKey() string     { return "OWNER" }
func (intentAliasKeys) GetHostCacheKey() string { return "legacy" }
func TestRefreshIntentRetirementIsAtomicAcrossAliases(t *testing.T) {
	c := intentFixture(t)
	old := intentToken("0")
	seedIntent(t, c, "OWNER", old)
	seedIntent(t, c, "legacy", old)
	wrapped := NewDualWritingStore(c, intentAliasKeys{})
	result := intentToken("1")
	_, err := WithFileTransaction(t.Context(), wrapped, func(view Store) error {
		a, err := BeginFileRefresh(t.Context(), view, "OWNER", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		if err := a.RecordResponse(result); err != nil {
			return err
		}
		return view.Put("OWNER", result)
	}, nil)
	if err != nil {
		t.Fatal("wrapped completion failed")
	}
	_, _ = WithFileTransaction(t.Context(), wrapped, func(view Store) error {
		_, err := BeginFileRefresh(t.Context(), view, "OWNER", sha256.Sum256([]byte("route")))
		if err != nil {
			return err
		}
		return errors.New("closed stop")
	}, nil)
	if err := DeleteWithContext(t.Context(), wrapped, "OWNER"); err != nil {
		t.Fatal("owner-first alias batch failed atomic retirement")
	}
	j := readIntent(t, c)
	g := j.generation(result)
	if j.Generations[g].Reservation != nil || !j.Generations[g].Consumed {
		t.Fatal("batch retirement lost consumed history or ownership")
	}
}

func TestRefreshIntentSmallHistoryAndPriorRecordChecks(t *testing.T) {
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	calls := 0
	for _, g := range []string{"1", "2", "3"} {
		if err := intentRefresh(t, c, "primary", intentToken(g), &calls); err != nil {
			t.Fatal("finite history failed")
		}
	}
	j := readIntent(t, c)
	if calls != 3 || j.sequence != 9 || len(j.Attempts) != 3 || len(j.Generations) != 4 {
		t.Fatal("finite replay lost state or duplicated transitions")
	}
	raw, err := os.ReadFile(c.fileLocation + refreshIntentSuffix)
	if err != nil {
		t.Fatal("history read failed")
	}
	key := j.key("primary")
	binding := j.Bindings[key]
	proof := refreshRetirement{Key: key, Generation: binding.Generation, Entry: binding.Entry, Epoch: nullable(binding.Epoch), BindingEntry: nullable(binding.Entry)}
	delta := refreshDelta{Sequence: j.sequence + 1, Transition: "retire_bindings", Retirements: []refreshRetirement{proof}, Generations: map[string]generationChange{}, Attempts: map[string]attemptChange{}, Bindings: map[string]bindingChange{key: {Expect: nullable(intentHash("wrong-prior-record"))}}}
	payload, err := json.Marshal(delta)
	if err != nil {
		t.Fatal("fixture encoding failed")
	}
	frame, _, err := encodeRefreshFrame(payload, j.chain[:], false)
	if err != nil {
		t.Fatal("fixture framing failed")
	}
	if decoded, err := decodeRefreshJournal(append(raw, frame...), c.fileLocation); decoded != nil || !errors.Is(err, ErrRefreshIntentInvalid) {
		t.Fatal("valid frame checksum bypassed prior-record expectation")
	}
}

func TestRefreshIntentLoginHeaderFailureKeepsCleanup(t *testing.T) {
	c := intentFixture(t)
	seedIntent(t, c, "primary", intentToken("0"))
	seedIntent(t, c, "unrelated", intentToken("9"))
	result := intentToken("1")
	_, err := WithFileTransaction(t.Context(), c, func(view Store) error {
		if err := RecordFileLoginResult(view, "primary", result); err != nil {
			return err
		}
		if err := view.Put("primary", result); err != nil {
			return err
		}
		return os.Mkdir(c.fileLocation+refreshIntentSuffix, 0o700)
	}, func(view Store) error {
		if err := os.Remove(c.fileLocation + refreshIntentSuffix); err != nil {
			return err
		}
		return view.Delete("primary")
	})
	if !errors.Is(err, ErrRefreshIntentIO) {
		t.Fatal("header failure incorrectly succeeded")
	}
	if _, err := c.Lookup("primary"); !errors.Is(err, ErrNotFound) {
		t.Fatal("header failure lost stale-token cleanup")
	}
	if _, err := c.Lookup("unrelated"); err != nil {
		t.Fatal("header failure changed unrelated entry")
	}
}

func TestRefreshIntentUnknownVersionAndNullRecordsFailClosed(t *testing.T) {
	for _, variant := range []string{"unknown-version", "null-generation", "null-attempt"} {
		t.Run(variant, func(t *testing.T) {
			c := intentFixture(t)
			seedIntent(t, c, "primary", intentToken("0"))
			j, err := newRefreshJournal(c.fileLocation)
			if err != nil {
				t.Fatal("header fixture failed")
			}
			header := refreshHeader{Version: 1, Scope: j.Scope, Nonce: j.Nonce}
			if variant == "unknown-version" {
				header.Version = 2
			}
			payload, err := json.Marshal(header)
			if err != nil {
				t.Fatal("header encoding failed")
			}
			raw, chain, err := encodeRefreshFrame(payload, []byte(refreshIntentMagic), true)
			if err != nil {
				t.Fatal("header framing failed")
			}
			if variant != "unknown-version" {
				nonce, err := intentNonce()
				if err != nil {
					t.Fatal("nonce fixture failed")
				}
				g := j.generation(intentToken("0"))
				delta := refreshDelta{Sequence: 1, Transition: "prepare_refresh", Attempt: nullable(nonce), Generations: map[string]generationChange{}, Bindings: map[string]bindingChange{}, Attempts: map[string]attemptChange{}}
				if variant == "null-generation" {
					delta.Generations[g] = generationChange{}
				} else {
					delta.Attempts[nonce] = attemptChange{}
				}
				payload, err = json.Marshal(delta)
				if err != nil {
					t.Fatal("delta encoding failed")
				}
				frame, _, err := encodeRefreshFrame(payload, chain[:], false)
				if err != nil {
					t.Fatal("delta framing failed")
				}
				raw = append(raw, frame...)
			}
			if err := os.WriteFile(c.fileLocation+refreshIntentSuffix, raw, 0o600); err != nil {
				t.Fatal("invalid-state fixture failed")
			}
			calls := 0
			_, err = WithFileTransaction(t.Context(), c, func(view Store) error { calls++; return nil }, nil)
			if !errors.Is(err, ErrRefreshIntentInvalid) || calls != 0 {
				t.Fatal("unknown or null intent authorized callback")
			}
		})
	}
}

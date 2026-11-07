package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

var (
	ErrRefreshRecoveryRequired = errors.New("cached OAuth refresh outcome is uncertain; automatic refresh is blocked")
	ErrRefreshIntentInvalid    = errors.New("cached OAuth refresh metadata is invalid; automatic refresh is blocked")
	ErrRefreshIntentIO         = errors.New("cannot persist OAuth refresh state")
	errRefreshOperation        = errors.New("invalid file refresh operation")
)

// FileRefreshStateError exposes a closed reason without rendering credential metadata.
type FileRefreshStateError struct{ reason, cause error }

func (e *FileRefreshStateError) Error() string {
	if e == nil || e.reason == nil {
		return errRefreshOperation.Error()
	}
	return e.reason.Error()
}

func (e *FileRefreshStateError) Unwrap() []error {
	if e == nil || e.reason == nil {
		return []error{errRefreshOperation}
	}
	if e.cause == nil {
		return []error{e.reason}
	}
	return []error{e.reason, e.cause}
}

func intentError(reason, cause error) error {
	return &FileRefreshStateError{reason: reason, cause: cause}
}

func intentHash(parts ...string) string {
	h := sha256.New()
	var size [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func intentNonce() (string, error) {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", intentError(ErrRefreshIntentIO, err)
	}
	return hex.EncodeToString(nonce[:]), nil
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type refreshGeneration struct {
	Consumed    bool    `json:"consumed"`
	ActiveEpoch *string `json:"active_epoch"`
	Reservation *string `json:"reservation"`
}
type refreshBinding struct {
	Generation string `json:"generation"`
	Epoch      string `json:"epoch"`
	Entry      string `json:"entry_d"`
}
type refreshWrite struct {
	Key                string  `json:"key_d"`
	BeforeSlot         string  `json:"before_slot_d"`
	BeforeGeneration   *string `json:"before_generation"`
	BeforeEpoch        *string `json:"before_epoch"`
	BeforeBindingEntry *string `json:"before_binding_entry_d"`
	BeforeReservation  *string `json:"before_reservation"`
	AfterEntry         string  `json:"after_entry_d"`
	AfterGeneration    *string `json:"after_generation"`
	AfterEpoch         *string `json:"after_epoch"`
}
type refreshAttempt struct {
	Kind             string         `json:"kind"`
	Phase            string         `json:"phase"`
	InputGeneration  *string        `json:"input_generation"`
	InputEpoch       *string        `json:"input_epoch"`
	InputKey         string         `json:"input_key_d"`
	InputEntry       *string        `json:"input_entry_d"`
	PreCache         string         `json:"pre_cache_d"`
	Request          *string        `json:"request_d"`
	ResponseEntry    *string        `json:"response_entry_d"`
	ResultGeneration *string        `json:"result_generation"`
	PostCache        *string        `json:"post_cache_d"`
	Writes           []refreshWrite `json:"writes"`
	Proof            *string        `json:"response_proof_d"`
}
type refreshJournal struct {
	retirements []refreshRetirement           `json:"-"`
	disk        *refreshJournal               `json:"-"`
	sequence    uint64                        `json:"-"`
	chain       [32]byte                      `json:"-"`
	end         int64                         `json:"-"`
	failed      error                         `json:"-"`
	Version     int                           `json:"version"`
	Scope       string                        `json:"store_scope_d"`
	Nonce       string                        `json:"store_nonce"`
	Generations map[string]*refreshGeneration `json:"generations"`
	Bindings    map[string]*refreshBinding    `json:"bindings"`
	Attempts    map[string]*refreshAttempt    `json:"attempts"`
}

func newRefreshJournal(location string) (*refreshJournal, error) {
	nonce, err := intentNonce()
	if err != nil {
		return nil, err
	}
	return &refreshJournal{
		Version: 1, Scope: intentHash("cli-file-intent/store/v1", location), Nonce: nonce,
		Generations: map[string]*refreshGeneration{}, Bindings: map[string]*refreshBinding{}, Attempts: map[string]*refreshAttempt{},
	}, nil
}

func (j *refreshJournal) key(key string) string {
	return intentHash("cli-file-intent/key/v1", j.Nonce, key)
}

func (j *refreshJournal) generation(entry Entry) string {
	if entry.Token == nil || entry.Token.RefreshToken == "" {
		return ""
	}
	return intentHash("cli-file-intent/generation/v1", j.Nonce, entry.Token.RefreshToken)
}

func entryIntentDigest(entry *fileEntry) (string, error) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return "", intentError(ErrRefreshIntentInvalid, err)
	}
	return intentHash("cli-file-intent/entry/v1", string(raw)), nil
}

func slotIntentDigest(file *tokenStoreFile, key string) (string, error) {
	entry, ok := file.Tokens[key]
	if !ok {
		return intentHash("cli-file-intent/slot/v1", "absent"), nil
	}
	digest, err := entryIntentDigest(entry)
	if err != nil {
		return "", err
	}
	return intentHash("cli-file-intent/slot/v1", "present", digest), nil
}

func cacheIntentDigest(raw []byte) string { return intentHash("cli-file-intent/cache/v1", string(raw)) }

func attemptProof(nonce string, a *refreshAttempt) string {
	copy := *a
	copy.Phase = ""
	copy.Proof = nil
	raw, _ := json.Marshal(copy)
	return intentHash("cli-file-intent/response/v1", nonce, string(raw))
}

// FileRefreshAttempt is valid only during its owning staged transaction callback.
type FileRefreshAttempt struct {
	view     *fileTransaction
	nonce    string
	key      string
	response *Entry
}

func transactionIntentView(view Store) *fileTransaction {
	switch view := view.(type) {
	case *fileTransaction:
		return view
	case *DualWritingStore:
		return transactionIntentView(view.inner)
	case *notFoundHintStore:
		return transactionIntentView(view.inner)
	default:
		return nil
	}
}

func (t *fileTransaction) ensureJournal() error {
	if t.journal != nil {
		return nil
	}
	journal, err := newRefreshJournal(t.owner.fileLocation)
	if err != nil {
		return err
	}
	t.journal = journal
	return nil
}

// BeginFileRefresh records consumption before the caller exchanges a cached credential.
// The view must come from the current WithFileTransaction callback.
func BeginFileRefresh(ctx context.Context, view Store, key string, requestDigest [32]byte) (*FileRefreshAttempt, error) {
	t := transactionIntentView(view)
	if t == nil || !t.live || t.attempt != nil {
		return nil, intentError(errRefreshOperation, nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := t.ctx.Err(); err != nil {
		return nil, err
	}
	current, err := t.Lookup(key)
	if err != nil {
		return nil, err
	}
	if current.Token == nil || current.Token.RefreshToken == "" {
		return nil, intentError(errRefreshOperation, nil)
	}
	if err := t.ensureJournal(); err != nil {
		return nil, err
	}
	j := t.journal
	g := j.generation(current)
	k := j.key(key)
	entry, err := entryIntentDigest(t.file.Tokens[key])
	if err != nil {
		return nil, err
	}
	gen := j.Generations[g]
	if gen != nil {
		binding := j.Bindings[k]
		if gen.Reservation != nil || gen.ActiveEpoch == nil || binding == nil || binding.Generation != g || binding.Epoch != deref(gen.ActiveEpoch) || binding.Entry != entry {
			return nil, intentError(ErrRefreshRecoveryRequired, nil)
		}
	} else {
		gen = &refreshGeneration{}
		j.Generations[g] = gen
	}
	nonce, err := intentNonce()
	if err != nil {
		return nil, err
	}
	a := &refreshAttempt{
		Kind: "refresh", Phase: "prepared", InputGeneration: nullable(g), InputEpoch: gen.ActiveEpoch,
		InputKey: k, InputEntry: nullable(entry), PreCache: cacheIntentDigest(t.originalRaw), Request: nullable(hex.EncodeToString(requestDigest[:])), Writes: []refreshWrite{},
	}
	j.Attempts[nonce] = a
	gen.Consumed = true
	gen.ActiveEpoch = nil
	gen.Reservation = nullable(nonce)
	if err := t.owner.saveRefreshJournal(t.ctx, j); err != nil {
		return nil, err
	}
	attempt := &FileRefreshAttempt{view: t, nonce: nonce, key: key}
	t.attempt = attempt
	return attempt, nil
}

// RecordResponse binds an actual successful exchange to this live attempt.
func (a *FileRefreshAttempt) RecordResponse(entry Entry) error {
	if a == nil || a.view == nil || !a.view.live || a.view.attempt != a || a.response != nil || entry.Token == nil || entry.Token.AccessToken == "" {
		return intentError(errRefreshOperation, nil)
	}
	if _, err := entryIntentDigest(&fileEntry{Token: entry.Token, PendingLogin: entry.PendingLogin}); err != nil {
		return err
	}
	copy := cloneEntry(entry)
	a.response = &copy
	return nil
}

// RecordFileLoginResult marks an actual fresh-login result without inventing a refresh generation.
func RecordFileLoginResult(view Store, key string, entry Entry) error {
	t := transactionIntentView(view)
	if t == nil || !t.live || t.attempt != nil || entry.Token == nil || entry.Token.AccessToken == "" {
		return intentError(errRefreshOperation, nil)
	}
	if _, err := entryIntentDigest(&fileEntry{Token: entry.Token, PendingLogin: entry.PendingLogin}); err != nil {
		return err
	}
	// Access-only login needs no companion until existing generation metadata exists.
	if t.journal == nil && entry.Token.RefreshToken == "" {
		return nil
	}
	if err := t.ensureJournal(); err != nil {
		return err
	}
	nonce, err := intentNonce()
	if err != nil {
		return err
	}
	prior, _ := t.Lookup(key)
	k := t.journal.key(key)
	var epoch *string
	if b := t.journal.Bindings[k]; b != nil {
		epoch = nullable(b.Epoch)
	}
	inputEntry, err := entryIntentDigest(t.file.Tokens[key])
	if err != nil {
		return err
	}
	a := &refreshAttempt{
		Kind: "login", Phase: "prepared", InputGeneration: nullable(t.journal.generation(prior)), InputEpoch: epoch,
		InputKey: k, InputEntry: nullable(inputEntry), PreCache: cacheIntentDigest(t.originalRaw), Writes: []refreshWrite{},
	}
	if _, present := t.file.Tokens[key]; !present {
		a.InputEntry = nil
	}
	t.journal.Attempts[nonce] = a
	copy := cloneEntry(entry)
	t.attempt = &FileRefreshAttempt{view: t, nonce: nonce, key: key, response: &copy}
	return nil
}

func (t *fileTransaction) responseProof(raw []byte) error {
	if t.attempt == nil {
		return nil
	}
	handle := t.attempt
	a := t.journal.Attempts[handle.nonce]
	if handle.response == nil {
		return intentError(errRefreshOperation, nil)
	}
	response := &fileEntry{Token: handle.response.Token, PendingLogin: handle.response.PendingLogin}
	responseEntry, err := entryIntentDigest(response)
	if err != nil {
		return err
	}
	primary, err := entryIntentDigest(t.file.Tokens[t.primaryKey()])
	if err != nil {
		return err
	}
	if primary != responseEntry {
		return intentError(errRefreshOperation, nil)
	}
	a.ResponseEntry = nullable(responseEntry)
	a.ResultGeneration = nullable(t.journal.generation(*handle.response))
	a.PostCache = nullable(cacheIntentDigest(raw))
	a.Writes = []refreshWrite{}
	keys := make([]string, 0, len(t.touched))
	for key := range t.touched {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		k := t.journal.key(key)
		old := t.original.Tokens[key]
		oldEntry := Entry{}
		if old != nil {
			oldEntry = Entry{Token: old.Token, PendingLogin: old.PendingLogin}
		}
		g := t.journal.generation(oldEntry)
		var epoch, reservation, bindingEntry *string
		if b := t.beforeBindings[k]; b != nil {
			epoch = nullable(b.Epoch)
			bindingEntry = nullable(b.Entry)
		}
		if gen := t.beforeGenerations[g]; gen != nil {
			reservation = gen.Reservation
		}
		after := t.file.Tokens[key]
		afterEntry := Entry{}
		if after != nil {
			afterEntry = Entry{Token: after.Token, PendingLogin: after.PendingLogin}
		}
		beforeSlot, err := slotIntentDigest(t.original, key)
		if err != nil {
			return err
		}
		afterDigest, err := entryIntentDigest(after)
		if err != nil {
			return err
		}
		w := refreshWrite{
			Key: k, BeforeSlot: beforeSlot, BeforeGeneration: nullable(g), BeforeEpoch: epoch, BeforeBindingEntry: bindingEntry, BeforeReservation: reservation,
			AfterEntry: afterDigest, AfterGeneration: nullable(t.journal.generation(afterEntry)),
		}
		if after != nil && responseEntry == w.AfterEntry && w.AfterGeneration != nil {
			w.AfterEpoch = nullable(handle.nonce)
		}
		a.Writes = append(a.Writes, w)
	}
	if !t.journal.canComplete(handle.nonce, a) {
		return intentError(ErrRefreshRecoveryRequired, nil)
	}
	a.Phase = "response_ready"
	a.Proof = nullable(attemptProof(handle.nonce, a))
	return t.owner.saveRefreshJournal(t.ctx, t.journal)
}

func (t *fileTransaction) primaryKey() string { return t.attempt.key }

func (j *refreshJournal) capturedReservation(a *refreshAttempt, g, nonce string) bool {
	owner := j.Attempts[nonce]
	if owner == nil || owner.Kind != "refresh" || deref(owner.InputGeneration) != g {
		return false
	}
	for _, w := range a.Writes {
		if w.Key == owner.InputKey && deref(w.BeforeGeneration) == g && deref(w.BeforeReservation) == nonce && deref(w.BeforeEpoch) == deref(owner.InputEpoch) && owner.InputEntry != nil && w.BeforeSlot == intentHash("cli-file-intent/slot/v1", "present", *owner.InputEntry) {
			return true
		}
	}
	return false
}

func (j *refreshJournal) canComplete(nonce string, a *refreshAttempt) bool {
	if a.Kind == "refresh" {
		gen := j.Generations[deref(a.InputGeneration)]
		if gen == nil || deref(gen.Reservation) != nonce {
			return false
		}
	}
	for _, w := range a.Writes {
		b := j.Bindings[w.Key]
		if b != nil && (b.Generation != deref(w.BeforeGeneration) || b.Epoch != deref(w.BeforeEpoch) || b.Entry != deref(w.BeforeBindingEntry)) {
			return false
		}
		if b == nil && (w.BeforeEpoch != nil || w.BeforeBindingEntry != nil) {
			return false
		}
		g := deref(w.BeforeGeneration)
		if g != "" {
			current := ""
			if gen := j.Generations[g]; gen != nil {
				current = deref(gen.Reservation)
			}
			if (a.Kind != "refresh" || g != deref(a.InputGeneration) || current != nonce) && current != deref(w.BeforeReservation) {
				return false
			}
			if a.Kind == "login" && current != "" {
				owner := j.Attempts[current]
				if owner != nil && owner.InputKey == w.Key && !j.capturedReservation(a, g, current) {
					return false
				}
			}
		}
	}
	result := deref(a.ResultGeneration)
	if result != "" {
		if gen := j.Generations[result]; gen != nil && gen.Reservation != nil && deref(gen.Reservation) != nonce {
			if a.Kind != "login" || !j.capturedReservation(a, result, *gen.Reservation) {
				return false
			}
		}
	}
	return true
}

func (j *refreshJournal) complete(nonce string, a *refreshAttempt) error {
	if !j.canComplete(nonce, a) {
		return intentError(ErrRefreshRecoveryRequired, nil)
	}
	if a.Kind == "refresh" {
		j.Generations[deref(a.InputGeneration)].Reservation = nil
	}
	if a.Kind == "login" {
		for _, w := range a.Writes {
			g := deref(w.BeforeGeneration)
			reservation := deref(w.BeforeReservation)
			if g != "" && reservation != "" && j.capturedReservation(a, g, reservation) {
				gen := j.Generations[g]
				if gen != nil && deref(gen.Reservation) == reservation {
					j.Attempts[reservation].Phase = "uncertain"
					gen.Reservation = nil
				}
			}
		}
	}
	for _, w := range a.Writes {
		delete(j.Bindings, w.Key)
	}
	if g := deref(a.ResultGeneration); g != "" {
		gen := j.Generations[g]
		if gen == nil {
			gen = &refreshGeneration{}
			j.Generations[g] = gen
		}
		if gen.Reservation != nil && deref(gen.Reservation) != nonce {
			return intentError(ErrRefreshRecoveryRequired, nil)
		}
		gen.Reservation = nil
		gen.ActiveEpoch = nullable(nonce)
		for _, w := range a.Writes {
			if w.AfterEpoch != nil && deref(w.AfterGeneration) == g {
				j.Bindings[w.Key] = &refreshBinding{Generation: g, Epoch: nonce, Entry: w.AfterEntry}
			}
		}
	}
	a.Phase = "completed"
	return nil
}

func (c *fileStore) reconcileRefreshJournal(ctx context.Context, j *refreshJournal, raw []byte) error {
	if j == nil {
		return nil
	}
	keys := make([]string, 0, len(j.Attempts))
	for nonce := range j.Attempts {
		keys = append(keys, nonce)
	}
	slices.Sort(keys)
	for _, nonce := range keys {
		a := j.Attempts[nonce]
		if a.Phase != "response_ready" {
			continue
		}
		if deref(a.PostCache) == cacheIntentDigest(raw) && j.canComplete(nonce, a) {
			if err := j.complete(nonce, a); err != nil {
				return err
			}
		} else {
			a.Phase = "uncertain"
		}
		if err := c.saveRefreshJournal(ctx, j); err != nil {
			return err
		}
	}
	return nil
}

func (t *fileTransaction) retireTouchedBindings() error {
	if t.journal == nil {
		return nil
	}
	t.journal.retirements = nil
	var proofs []refreshRetirement
	keys := make([]string, 0, len(t.touched))
	for key := range t.touched {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		oldSlot, err := slotIntentDigest(t.original, key)
		if err != nil {
			return err
		}
		newSlot, err := slotIntentDigest(t.file, key)
		if err != nil {
			return err
		}
		if oldSlot == newSlot {
			continue
		}
		entry := t.original.Tokens[key]
		if entry == nil || entry.Token == nil || entry.RefreshToken == "" {
			continue
		}
		k := t.journal.key(key)
		g := t.journal.generation(Entry{Token: entry.Token, PendingLogin: entry.PendingLogin})
		digest, err := entryIntentDigest(entry)
		if err != nil {
			return err
		}
		proof := refreshRetirement{Key: k, Generation: g, Entry: digest}
		if b := t.beforeBindings[k]; b != nil {
			proof.Epoch = nullable(b.Epoch)
			proof.BindingEntry = nullable(b.Entry)
		}
		if gen := t.beforeGenerations[g]; gen != nil {
			proof.Reservation = gen.Reservation
		}
		proofs = append(proofs, proof)
	}
	used, err := t.journal.retireCapturedSet(proofs)
	if err != nil {
		return err
	}
	t.journal.retirements = used
	return nil
}

func (t *fileTransaction) finishJournal() error {
	if t.journal == nil {
		return nil
	}
	if t.attempt != nil {
		a := t.journal.Attempts[t.attempt.nonce]
		if err := t.journal.complete(t.attempt.nonce, a); err != nil {
			return err
		}
	} else {
		if err := t.retireTouchedBindings(); err != nil {
			return err
		}
	}
	return t.owner.saveRefreshJournal(t.ctx, t.journal)
}

func (t *fileTransaction) abandonPublication() error {
	if t.attempt == nil {
		return nil
	}
	if t.journal.failed != nil {
		return intentError(ErrRefreshIntentIO, t.journal.failed)
	}
	if t.journal.disk == nil {
		delete(t.journal.Attempts, t.attempt.nonce)
		return nil
	}
	prior := t.journal.disk.Attempts[t.attempt.nonce]
	if prior == nil {
		delete(t.journal.Attempts, t.attempt.nonce)
		return nil
	}
	if prior.Phase == "uncertain" {
		return nil
	}
	copy := *prior
	copy.Phase = "uncertain"
	t.journal.Attempts[t.attempt.nonce] = &copy
	return t.owner.saveRefreshJournal(t.ctx, t.journal)
}

func copyBindings(j *refreshJournal) map[string]*refreshBinding {
	result := map[string]*refreshBinding{}
	if j != nil {
		for k, b := range j.Bindings {
			copy := *b
			result[k] = &copy
		}
	}
	return result
}

func copyGenerations(j *refreshJournal) map[string]*refreshGeneration {
	result := map[string]*refreshGeneration{}
	if j != nil {
		for k, g := range j.Generations {
			copy := *g
			result[k] = &copy
		}
	}
	return result
}

func newFileTransaction(ctx context.Context, c *fileStore, file *tokenStoreFile, raw []byte, j *refreshJournal) *fileTransaction {
	return &fileTransaction{
		owner: c, ctx: ctx, file: cloneStoreFile(file), original: cloneStoreFile(file), originalRaw: raw, journal: j,
		touched: map[string]bool{}, live: true, beforeBindings: copyBindings(j), beforeGenerations: copyGenerations(j),
	}
}

// RefreshRequestDigest binds evidence to the actual configured refresh route.
func RefreshRequestDigest(endpoint, clientID string) [32]byte {
	text := intentHash("cli-file-intent/request/v1", endpoint, clientID)
	bytes, _ := hex.DecodeString(text)
	var result [32]byte
	copy(result[:], bytes)
	return result
}

func intentStateReason(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("cache update: %w", err)
}

func (c *fileStore) publishFileTransaction(t *fileTransaction, cleanup func(Store) error) error {
	if !t.dirty {
		if t.attempt != nil {
			return intentError(errRefreshOperation, nil)
		}
		return nil
	}
	raw, err := encodeTokenStoreFile(t.file)
	if err != nil {
		return c.failedFilePublication(t, cleanup, err)
	}
	if t.attempt != nil {
		if err := t.responseProof(raw); err != nil {
			return c.failedFilePublication(t, cleanup, err)
		}
	} else if t.journal != nil {
		if err := t.retireTouchedBindings(); err != nil {
			return c.failedFilePublication(t, cleanup, err)
		}
		if err := c.saveRefreshJournal(t.ctx, t.journal); err != nil {
			return c.failedFilePublication(t, cleanup, err)
		}
	}
	if err := c.writeTokenStoreBytes(raw); err != nil {
		return c.failedFilePublication(t, cleanup, err)
	}
	if t.attempt != nil {
		if c.beforeIntentCompletion != nil {
			c.beforeIntentCompletion()
		}
		return t.finishJournal()
	}
	return nil
}

func (c *fileStore) failedFilePublication(t *fileTransaction, cleanup func(Store) error, cause error) error {
	publicationErr := intentStateReason(cause)
	stateErr := t.abandonPublication()
	if cleanup == nil {
		return errors.Join(publicationErr, stateErr)
	}
	rollback := newFileTransaction(t.ctx, c, t.original, t.originalRaw, t.journal)
	defer func() { rollback.live = false }()
	cleanupErr := cleanup(rollback)
	rollback.live = false
	if cleanupErr == nil && rollback.dirty {
		// An uncertain append is never followed by another journal append.
		// Existing failure cleanup may still remove the stale cache entries.
		if rollback.journal != nil && rollback.journal.failed == nil && rollback.journal.disk != nil {
			cleanupErr = rollback.retireTouchedBindings()
			if cleanupErr == nil {
				cleanupErr = c.saveRefreshJournal(t.ctx, rollback.journal)
			}
		}
		if cleanupErr == nil {
			cleanupErr = c.write(rollback.file)
		}
	}
	return errors.Join(publicationErr, stateErr, cleanupErr)
}

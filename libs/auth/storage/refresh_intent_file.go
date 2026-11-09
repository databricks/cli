package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
)

const (
	refreshIntentMagic  = "CLIRFI1\n"
	refreshIntentSuffix = ".refresh-intents.json"
)

type refreshHeader struct {
	Version int    `json:"version"`
	Scope   string `json:"store_scope_d"`
	Nonce   string `json:"store_nonce"`
}
type generationChange struct {
	Expect *string            `json:"expect_d"`
	Value  *refreshGeneration `json:"value"`
}
type bindingChange struct {
	Expect *string         `json:"expect_d"`
	Value  *refreshBinding `json:"value"`
}
type attemptChange struct {
	Expect *string         `json:"expect_d"`
	Value  *refreshAttempt `json:"value"`
}
type refreshDelta struct {
	Retirements []refreshRetirement         `json:"retirements,omitempty"`
	Sequence    uint64                      `json:"seq"`
	Transition  string                      `json:"transition"`
	Attempt     *string                     `json:"attempt_nonce"`
	Generations map[string]generationChange `json:"generations"`
	Bindings    map[string]bindingChange    `json:"bindings"`
	Attempts    map[string]attemptChange    `json:"attempts"`
}

func digestValid(s string) bool {
	if len(s) != 64 {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && hex.EncodeToString(b) == s
}
func optionalValid(s *string) bool    { return s == nil || digestValid(*s) }
func invalidIntent(cause error) error { return intentError(ErrRefreshIntentInvalid, cause) }
func canonicalDecode(raw []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return invalidIntent(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return invalidIntent(nil)
	}
	encoded, err := json.Marshal(value)
	if err != nil || !bytes.Equal(raw, encoded) {
		return invalidIntent(err)
	}
	return nil
}

func cloneRefreshJournal(j *refreshJournal) (*refreshJournal, error) {
	if j == nil {
		return nil, invalidIntent(nil)
	}
	clone := &refreshJournal{Version: j.Version, Scope: j.Scope, Nonce: j.Nonce, Generations: map[string]*refreshGeneration{}, Bindings: map[string]*refreshBinding{}, Attempts: map[string]*refreshAttempt{}}
	for k, v := range j.Generations {
		if v == nil {
			return nil, invalidIntent(nil)
		}
		copy := *v
		clone.Generations[k] = &copy
	}
	for k, v := range j.Bindings {
		if v == nil {
			return nil, invalidIntent(nil)
		}
		copy := *v
		clone.Bindings[k] = &copy
	}
	for k, v := range j.Attempts {
		if v == nil {
			return nil, invalidIntent(nil)
		}
		cloned := *v
		if v.Writes != nil {
			cloned.Writes = make([]refreshWrite, len(v.Writes))
			copy(cloned.Writes, v.Writes)
		}
		clone.Attempts[k] = &cloned
	}
	return clone, nil
}

func recordDigest(kind, key string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", invalidIntent(err)
	}
	h := sha256.New()
	_, _ = h.Write([]byte("cli-file-intent/record/v1\x00"))
	for _, s := range []string{kind, key} {
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(s)))
		_, _ = h.Write(length[:])
		_, _ = h.Write([]byte(s))
	}
	_, _ = h.Write(raw)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func expectedRecord(kind, key string, value any) (*string, error) {
	if reflect.ValueOf(value).IsNil() {
		return nil, nil
	}
	digest, err := recordDigest(kind, key, value)
	if err != nil {
		return nil, err
	}
	return nullable(digest), nil
}

func expectationMatches(kind, key string, value any, expect *string) bool {
	digest, err := expectedRecord(kind, key, value)
	return err == nil && reflect.DeepEqual(digest, expect)
}

func validAttempt(nonce string, a *refreshAttempt) bool {
	if a == nil || !digestValid(nonce) || (a.Kind != "refresh" && a.Kind != "login") || !digestValid(a.InputKey) || !digestValid(a.PreCache) {
		return false
	}
	for _, v := range []*string{a.InputGeneration, a.InputEpoch, a.InputEntry, a.Request, a.ResponseEntry, a.ResultGeneration, a.PostCache, a.Proof} {
		if !optionalValid(v) {
			return false
		}
	}
	if a.Kind == "refresh" && (a.InputGeneration == nil || a.InputEntry == nil || a.Request == nil) {
		return false
	}
	if a.Kind == "login" && a.Request != nil {
		return false
	}
	switch a.Phase {
	case "prepared", "response_ready", "completed", "uncertain":
	default:
		return false
	}
	if a.Proof == nil {
		return a.Kind == "refresh" && (a.Phase == "prepared" || a.Phase == "uncertain") && a.ResponseEntry == nil && a.ResultGeneration == nil && a.PostCache == nil && len(a.Writes) == 0
	}
	if a.Phase == "prepared" || a.ResponseEntry == nil || a.PostCache == nil || len(a.Writes) == 0 || deref(a.Proof) != attemptProof(nonce, a) {
		return false
	}
	seen := map[string]bool{}
	primary := false
	for _, w := range a.Writes {
		if !digestValid(w.Key) || !digestValid(w.BeforeSlot) || !digestValid(w.AfterEntry) || seen[w.Key] {
			return false
		}
		seen[w.Key] = true
		for _, v := range []*string{w.BeforeGeneration, w.BeforeEpoch, w.BeforeBindingEntry, w.BeforeReservation, w.AfterGeneration, w.AfterEpoch} {
			if !optionalValid(v) {
				return false
			}
		}
		if (w.BeforeEpoch == nil) != (w.BeforeBindingEntry == nil) || (w.BeforeEpoch != nil && w.BeforeGeneration == nil) || (w.BeforeReservation != nil && w.BeforeGeneration == nil) {
			return false
		}
		if (w.AfterGeneration == nil) != (w.AfterEpoch == nil) {
			return false
		}
		if w.AfterEpoch != nil && (deref(w.AfterEpoch) != nonce || deref(w.AfterGeneration) != deref(a.ResultGeneration) || w.AfterEntry != deref(a.ResponseEntry)) {
			return false
		}
		if w.Key == a.InputKey && w.AfterEntry == deref(a.ResponseEntry) {
			primary = true
		}
	}
	return primary
}

func validateRefreshJournal(j *refreshJournal) error {
	if j == nil || j.Version != 1 || !digestValid(j.Scope) || !digestValid(j.Nonce) || j.Generations == nil || j.Bindings == nil || j.Attempts == nil {
		return invalidIntent(nil)
	}
	for nonce, a := range j.Attempts {
		if !validAttempt(nonce, a) {
			return invalidIntent(nil)
		}
	}
	for g, v := range j.Generations {
		if !digestValid(g) || v == nil || !optionalValid(v.ActiveEpoch) || !optionalValid(v.Reservation) || (v.ActiveEpoch != nil && v.Reservation != nil) {
			return invalidIntent(nil)
		}
		if v.ActiveEpoch != nil {
			a := j.Attempts[*v.ActiveEpoch]
			if a == nil || a.Phase != "completed" || deref(a.ResultGeneration) != g {
				return invalidIntent(nil)
			}
		}
		if v.Reservation != nil {
			a := j.Attempts[*v.Reservation]
			if !v.Consumed || a == nil || a.Kind != "refresh" || deref(a.InputGeneration) != g || a.Phase == "completed" {
				return invalidIntent(nil)
			}
		}
	}
	indexes := map[string]map[string]refreshWrite{}
	for k, b := range j.Bindings {
		if !digestValid(k) || b == nil || !digestValid(b.Generation) || !digestValid(b.Epoch) || !digestValid(b.Entry) || j.Generations[b.Generation] == nil {
			return invalidIntent(nil)
		}
		a := j.Attempts[b.Epoch]
		if a == nil || a.Phase != "completed" {
			return invalidIntent(nil)
		}
		index := indexes[b.Epoch]
		if index == nil {
			index = map[string]refreshWrite{}
			for _, w := range a.Writes {
				index[w.Key] = w
			}
			indexes[b.Epoch] = index
		}
		w, found := index[k]
		if !found || deref(w.AfterGeneration) != b.Generation || deref(w.AfterEpoch) != b.Epoch || w.AfterEntry != b.Entry {
			return invalidIntent(nil)
		}
	}
	return nil
}

func sameRefreshState(a, b *refreshJournal) bool {
	return a.Version == b.Version && a.Scope == b.Scope && a.Nonce == b.Nonce && reflect.DeepEqual(a.Generations, b.Generations) && reflect.DeepEqual(a.Bindings, b.Bindings) && reflect.DeepEqual(a.Attempts, b.Attempts)
}

func skeleton(a *refreshAttempt) refreshAttempt {
	copy := *a
	copy.Phase = ""
	copy.ResponseEntry = nil
	copy.ResultGeneration = nil
	copy.PostCache = nil
	copy.Writes = nil
	copy.Proof = nil
	return copy
}

func applyRefreshDelta(old *refreshJournal, d *refreshDelta) (*refreshJournal, error) {
	if err := validateRefreshDelta(old, d); err != nil {
		return nil, err
	}
	// A failed load is discarded; all checks precede these map updates.
	for k, c := range d.Generations {
		old.Generations[k] = c.Value
	}
	for k, c := range d.Bindings {
		if c.Value == nil {
			delete(old.Bindings, k)
		} else {
			old.Bindings[k] = c.Value
		}
	}
	for k, c := range d.Attempts {
		old.Attempts[k] = c.Value
	}
	return old, nil
}

func makeRefreshDelta(old, next *refreshJournal, seq uint64) (*refreshDelta, error) {
	d := &refreshDelta{Sequence: seq, Retirements: next.retirements, Generations: map[string]generationChange{}, Bindings: map[string]bindingChange{}, Attempts: map[string]attemptChange{}}
	for k := range old.Generations {
		if next.Generations[k] == nil {
			return nil, invalidIntent(nil)
		}
	}
	for k := range old.Attempts {
		if next.Attempts[k] == nil {
			return nil, invalidIntent(nil)
		}
	}
	for k, v := range next.Generations {
		if !reflect.DeepEqual(old.Generations[k], v) {
			e, err := expectedRecord("generation", k, old.Generations[k])
			if err != nil {
				return nil, err
			}
			d.Generations[k] = generationChange{Expect: e, Value: v}
		}
	}
	for k, v := range next.Attempts {
		if !reflect.DeepEqual(old.Attempts[k], v) {
			e, err := expectedRecord("attempt", k, old.Attempts[k])
			if err != nil {
				return nil, err
			}
			d.Attempts[k] = attemptChange{Expect: e, Value: v}
		}
	}
	keys := map[string]bool{}
	for k := range old.Bindings {
		keys[k] = true
	}
	for k := range next.Bindings {
		keys[k] = true
	}
	for k := range keys {
		if !reflect.DeepEqual(old.Bindings[k], next.Bindings[k]) {
			e, err := expectedRecord("binding", k, old.Bindings[k])
			if err != nil {
				return nil, err
			}
			d.Bindings[k] = bindingChange{Expect: e, Value: next.Bindings[k]}
		}
	}
	if len(d.Generations)+len(d.Bindings)+len(d.Attempts) == 0 {
		return nil, nil
	}
	if len(d.Retirements) != 0 {
		d.Transition = "retire_bindings"
	}
	for k, c := range d.Attempts {
		if d.Transition == "retire_bindings" {
			break
		}
		if c.Value.Phase == "completed" {
			d.Transition = "complete"
			d.Attempt = nullable(k)
		}
	}
	if d.Transition == "" {
		if len(d.Attempts) == 0 {
			d.Transition = "retire_bindings"
		} else if len(d.Attempts) == 1 {
			for k, c := range d.Attempts {
				d.Attempt = nullable(k)
				switch c.Value.Phase {
				case "prepared":
					d.Transition = "prepare_refresh"
				case "response_ready":
					d.Transition = "response_ready"
				case "uncertain":
					d.Transition = "uncertain"
				}
			}
		}
	}
	if err := validateRefreshDelta(old, d); err != nil {
		return nil, err
	}
	return d, nil
}

func refreshFrameHash(domain string, prior, length, payload []byte) [32]byte {
	h := sha256.New()
	_, _ = h.Write([]byte(domain))
	_, _ = h.Write(prior)
	_, _ = h.Write(length)
	_, _ = h.Write(payload)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func encodeRefreshFrame(payload, prior []byte, header bool) ([]byte, [32]byte, error) {
	if uint64(len(payload)) > math.MaxUint32 {
		return nil, [32]byte{}, invalidIntent(nil)
	}
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(payload)))
	domain := "cli-file-intent/delta/v1\x00"
	var prefix []byte
	if header {
		domain = "cli-file-intent/header/v1\x00"
		prefix = []byte(refreshIntentMagic)
	}
	digest := refreshFrameHash(domain, prior, length[:], payload)
	result := append(prefix, length[:]...)
	result = append(result, payload...)
	result = append(result, digest[:]...)
	return result, digest, nil
}

func openRegularRefreshFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, invalidIntent(nil)
	}
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, invalidIntent(err)
	}
	return file, nil
}

func (c *fileStore) loadRefreshJournal(ctx context.Context) (*refreshJournal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := openRegularRefreshFile(c.fileLocation+refreshIntentSuffix, os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		if errors.Is(err, ErrRefreshIntentInvalid) {
			return nil, err
		}
		return nil, intentError(ErrRefreshIntentIO, err)
	}
	raw, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, intentError(ErrRefreshIntentIO, errors.Join(readErr, closeErr))
	}
	return decodeRefreshJournal(raw, c.fileLocation)
}

func decodeRefreshJournal(raw []byte, location string) (*refreshJournal, error) {
	if len(raw) < len(refreshIntentMagic) || string(raw[:len(refreshIntentMagic)]) != refreshIntentMagic {
		return nil, invalidIntent(nil)
	}
	offset := len(refreshIntentMagic)
	readFrame := func(prior []byte, header bool) ([]byte, [32]byte, error) {
		if len(raw)-offset < 4 {
			return nil, [32]byte{}, invalidIntent(nil)
		}
		length := binary.BigEndian.Uint32(raw[offset : offset+4])
		start := offset
		offset += 4
		if uint64(length)+32 > uint64(len(raw)-offset) {
			return nil, [32]byte{}, invalidIntent(nil)
		}
		payload := raw[offset : offset+int(length)]
		offset += int(length)
		var got [32]byte
		copy(got[:], raw[offset:offset+32])
		offset += 32
		domain := "cli-file-intent/delta/v1\x00"
		if header {
			domain = "cli-file-intent/header/v1\x00"
		}
		want := refreshFrameHash(domain, prior, raw[start:start+4], payload)
		if got != want {
			return nil, [32]byte{}, invalidIntent(nil)
		}
		return payload, got, nil
	}
	payload, chain, err := readFrame([]byte(refreshIntentMagic), true)
	if err != nil {
		return nil, err
	}
	var header refreshHeader
	if err := canonicalDecode(payload, &header); err != nil {
		return nil, err
	}
	if header.Version != 1 || header.Scope != intentHash("cli-file-intent/store/v1", location) || !digestValid(header.Nonce) {
		return nil, invalidIntent(nil)
	}
	j := &refreshJournal{Version: 1, Scope: header.Scope, Nonce: header.Nonce, Generations: map[string]*refreshGeneration{}, Bindings: map[string]*refreshBinding{}, Attempts: map[string]*refreshAttempt{}}
	var sequence uint64
	for offset < len(raw) {
		payload, nextChain, err := readFrame(chain[:], false)
		if err != nil {
			return nil, err
		}
		var delta refreshDelta
		if err := canonicalDecode(payload, &delta); err != nil {
			return nil, err
		}
		if sequence == math.MaxUint64 || delta.Sequence != sequence+1 {
			return nil, invalidIntent(nil)
		}
		next, err := applyRefreshDelta(j, &delta)
		if err != nil {
			return nil, err
		}
		j = next
		sequence = delta.Sequence
		chain = nextChain
	}
	if err := validateRefreshJournal(j); err != nil {
		return nil, err
	}
	disk, err := cloneRefreshJournal(j)
	if err != nil {
		return nil, err
	}
	j.disk = disk
	j.sequence = sequence
	j.chain = chain
	j.end = int64(len(raw))
	return j, nil
}

func checkedRefreshWrite(file *os.File, raw []byte) error {
	n, err := file.Write(raw)
	if err != nil {
		return err
	}
	if n != len(raw) {
		return io.ErrShortWrite
	}
	return nil
}

func (c *fileStore) initializeRefreshJournal(ctx context.Context, j *refreshJournal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	header := refreshHeader{Version: 1, Scope: j.Scope, Nonce: j.Nonce}
	payload, err := json.Marshal(header)
	if err != nil {
		return invalidIntent(err)
	}
	raw, chain, err := encodeRefreshFrame(payload, []byte(refreshIntentMagic), true)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(c.fileLocation+refreshIntentSuffix, os.O_RDWR|os.O_CREATE|os.O_EXCL, ownerReadWrite)
	if err != nil {
		return intentError(ErrRefreshIntentIO, err)
	}
	err = checkedRefreshWrite(file, raw)
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		j.failed = err
		return intentError(ErrRefreshIntentIO, err)
	}
	j.chain = chain
	j.end = int64(len(raw))
	j.disk = &refreshJournal{Version: 1, Scope: j.Scope, Nonce: j.Nonce, Generations: map[string]*refreshGeneration{}, Bindings: map[string]*refreshBinding{}, Attempts: map[string]*refreshAttempt{}}
	return nil
}

func (c *fileStore) saveRefreshJournal(ctx context.Context, j *refreshJournal) error {
	if j.failed != nil {
		return intentError(ErrRefreshIntentIO, j.failed)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if j.disk == nil {
		if err := c.initializeRefreshJournal(ctx, j); err != nil {
			return err
		}
	}
	if j.sequence == math.MaxUint64 {
		return invalidIntent(nil)
	}
	delta, err := makeRefreshDelta(j.disk, j, j.sequence+1)
	if err != nil {
		return err
	}
	if delta == nil {
		return nil
	}
	payload, err := json.Marshal(delta)
	if err != nil {
		return invalidIntent(err)
	}
	raw, chain, err := encodeRefreshFrame(payload, j.chain[:], false)
	if err != nil {
		return err
	}
	file, err := openRegularRefreshFile(c.fileLocation+refreshIntentSuffix, os.O_RDWR, 0)
	if err != nil {
		return intentError(ErrRefreshIntentIO, err)
	}
	// Revalidate the complete file before appending; no valid-prefix authorization.
	existing, readErr := io.ReadAll(file)
	if readErr != nil {
		closeErr := file.Close()
		return intentError(ErrRefreshIntentIO, errors.Join(readErr, closeErr))
	}
	loaded, decodeErr := decodeRefreshJournal(existing, c.fileLocation)
	if decodeErr != nil {
		_ = file.Close()
		return decodeErr
	}
	if loaded == nil || loaded.end != j.end || loaded.sequence != j.sequence || loaded.chain != j.chain || !sameRefreshState(loaded, j.disk) {
		_ = file.Close()
		return invalidIntent(nil)
	}
	end, err := file.Seek(0, io.SeekEnd)
	if err != nil || end != j.end {
		_ = file.Close()
		return intentError(ErrRefreshIntentIO, err)
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		return err
	}
	err = checkedRefreshWrite(file, raw)
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		j.failed = err
		return intentError(ErrRefreshIntentIO, err)
	}
	disk, err := cloneRefreshJournal(j)
	if err != nil {
		j.failed = err
		return err
	}
	j.disk = disk
	j.retirements = nil
	j.sequence++
	j.chain = chain
	j.end += int64(len(raw))
	return nil
}

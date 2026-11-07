package storage

import "reflect"

// refreshRetirement is the exact leased snapshot of an entry being replaced.
type refreshRetirement struct {
	Key          string  `json:"key_d"`
	Generation   string  `json:"generation"`
	Entry        string  `json:"entry_d"`
	Epoch        *string `json:"epoch"`
	BindingEntry *string `json:"binding_entry_d"`
	Reservation  *string `json:"reservation"`
}

func (j *refreshJournal) checkRetirement(p refreshRetirement) (bool, bool, error) {
	if !digestValid(p.Key) || !digestValid(p.Generation) || !digestValid(p.Entry) || !optionalValid(p.Epoch) || !optionalValid(p.BindingEntry) || !optionalValid(p.Reservation) || (p.Epoch == nil) != (p.BindingEntry == nil) {
		return false, false, invalidIntent(nil)
	}
	binding := j.Bindings[p.Key]
	if binding != nil {
		if binding.Generation != p.Generation || binding.Epoch != deref(p.Epoch) || binding.Entry != deref(p.BindingEntry) || binding.Entry != p.Entry {
			return false, false, intentError(ErrRefreshRecoveryRequired, nil)
		}
	} else if p.Epoch != nil {
		return false, false, intentError(ErrRefreshRecoveryRequired, nil)
	}
	gen := j.Generations[p.Generation]
	reservation := ""
	if gen != nil {
		reservation = deref(gen.Reservation)
	}
	if reservation != deref(p.Reservation) {
		return false, false, intentError(ErrRefreshRecoveryRequired, nil)
	}
	clear := false
	if reservation != "" {
		owner := j.Attempts[reservation]
		if owner == nil || deref(owner.InputGeneration) != p.Generation {
			return false, false, invalidIntent(nil)
		}
		if owner.InputKey == p.Key {
			if deref(owner.InputEntry) != p.Entry || deref(owner.InputEpoch) != deref(p.Epoch) {
				return false, false, intentError(ErrRefreshRecoveryRequired, nil)
			}
			clear = true
		}
	}
	return binding != nil, clear, nil
}

func (j *refreshJournal) retireCapturedSet(proofs []refreshRetirement) ([]refreshRetirement, error) {
	type retirement struct {
		proof       refreshRetirement
		drop, clear bool
	}
	var actions []retirement
	var used []refreshRetirement
	seen := map[string]bool{}
	// Every expectation is checked against the same pre-transition state.
	for _, p := range proofs {
		if seen[p.Key] {
			return nil, invalidIntent(nil)
		}
		seen[p.Key] = true
		drop, clear, err := j.checkRetirement(p)
		if err != nil {
			return nil, err
		}
		if drop || clear {
			actions = append(actions, retirement{p, drop, clear})
			used = append(used, p)
		}
	}
	for _, a := range actions {
		if a.drop {
			delete(j.Bindings, a.proof.Key)
		}
		if a.clear {
			reservation := deref(a.proof.Reservation)
			j.Attempts[reservation].Phase = "uncertain"
			j.Generations[a.proof.Generation].Reservation = nil
		}
	}
	return used, nil
}

func (j *refreshJournal) retireCaptured(p refreshRetirement) (bool, error) {
	used, err := j.retireCapturedSet([]refreshRetirement{p})
	return len(used) != 0, err
}

// Only records referenced by this transition are copied for semantic validation.
func selectRefreshState(old *refreshJournal, a *refreshAttempt, retire []refreshRetirement, nonce string) *refreshJournal {
	view := &refreshJournal{Version: old.Version, Scope: old.Scope, Nonce: old.Nonce, Generations: map[string]*refreshGeneration{}, Bindings: map[string]*refreshBinding{}, Attempts: map[string]*refreshAttempt{}}
	takeAttempt := func(key string) {
		if value := old.Attempts[key]; value != nil {
			copy := *value
			view.Attempts[key] = &copy
		}
	}
	takeGeneration := func(key string) {
		if value := old.Generations[key]; value != nil {
			copy := *value
			view.Generations[key] = &copy
			takeAttempt(deref(value.Reservation))
		}
	}
	takeBinding := func(key string) {
		if value := old.Bindings[key]; value != nil {
			copy := *value
			view.Bindings[key] = &copy
		}
	}
	if a != nil {
		takeAttempt(nonce)
		takeGeneration(deref(a.InputGeneration))
		takeGeneration(deref(a.ResultGeneration))
		takeBinding(a.InputKey)
		for _, w := range a.Writes {
			takeGeneration(deref(w.BeforeGeneration))
			takeGeneration(deref(w.AfterGeneration))
			takeBinding(w.Key)
		}
	}
	for _, r := range retire {
		takeGeneration(r.Generation)
		takeBinding(r.Key)
	}
	return view
}

func selectedChanges(old, view *refreshJournal, gkeys, bkeys, akeys map[string]bool) (*refreshDelta, error) {
	d := &refreshDelta{Generations: map[string]generationChange{}, Bindings: map[string]bindingChange{}, Attempts: map[string]attemptChange{}}
	for k := range view.Generations {
		gkeys[k] = true
	}
	for k := range view.Bindings {
		bkeys[k] = true
	}
	for k := range view.Attempts {
		akeys[k] = true
	}
	for k := range gkeys {
		if !reflect.DeepEqual(old.Generations[k], view.Generations[k]) {
			e, err := expectedRecord("generation", k, old.Generations[k])
			if err != nil {
				return nil, err
			}
			d.Generations[k] = generationChange{Expect: e, Value: view.Generations[k]}
		}
	}
	for k := range bkeys {
		if !reflect.DeepEqual(old.Bindings[k], view.Bindings[k]) {
			e, err := expectedRecord("binding", k, old.Bindings[k])
			if err != nil {
				return nil, err
			}
			d.Bindings[k] = bindingChange{Expect: e, Value: view.Bindings[k]}
		}
	}
	for k := range akeys {
		if !reflect.DeepEqual(old.Attempts[k], view.Attempts[k]) {
			e, err := expectedRecord("attempt", k, old.Attempts[k])
			if err != nil {
				return nil, err
			}
			d.Attempts[k] = attemptChange{Expect: e, Value: view.Attempts[k]}
		}
	}
	return d, nil
}

func selectedKeys(j *refreshJournal) (map[string]bool, map[string]bool, map[string]bool) {
	g, b, a := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for k := range j.Generations {
		g[k] = true
	}
	for k := range j.Bindings {
		b[k] = true
	}
	for k := range j.Attempts {
		a[k] = true
	}
	return g, b, a
}

func expectedRefreshChanges(old *refreshJournal, d *refreshDelta) (*refreshDelta, error) {
	nonce := deref(d.Attempt)
	changed := d.Attempts[nonce].Value
	expected := &refreshDelta{Generations: map[string]generationChange{}, Bindings: map[string]bindingChange{}, Attempts: map[string]attemptChange{}}
	switch d.Transition {
	case "prepare_refresh":
		if changed == nil || old.Attempts[nonce] != nil || changed.Kind != "refresh" || changed.Phase != "prepared" {
			return nil, invalidIntent(nil)
		}
		g := deref(changed.InputGeneration)
		prior := old.Generations[g]
		gen := &refreshGeneration{}
		if prior == nil {
			if changed.InputEpoch != nil {
				return nil, invalidIntent(nil)
			}
		} else {
			*gen = *prior
			b := old.Bindings[changed.InputKey]
			if prior.Reservation != nil || prior.ActiveEpoch == nil || deref(prior.ActiveEpoch) != deref(changed.InputEpoch) || b == nil || b.Generation != g || b.Epoch != deref(changed.InputEpoch) || b.Entry != deref(changed.InputEntry) {
				return nil, invalidIntent(nil)
			}
		}
		gen.Consumed = true
		gen.ActiveEpoch = nil
		gen.Reservation = nullable(nonce)
		e, err := expectedRecord("generation", g, prior)
		if err != nil {
			return nil, err
		}
		expected.Generations[g] = generationChange{Expect: e, Value: gen}
		expected.Attempts[nonce] = attemptChange{Value: changed}
	case "response_ready":
		if changed == nil || changed.Phase != "response_ready" {
			return nil, invalidIntent(nil)
		}
		prior := old.Attempts[nonce]
		if changed.Kind == "refresh" {
			if prior == nil || prior.Phase != "prepared" || !reflect.DeepEqual(skeleton(prior), skeleton(changed)) {
				return nil, invalidIntent(nil)
			}
		} else if prior != nil {
			return nil, invalidIntent(nil)
		}
		if !old.canComplete(nonce, changed) {
			return nil, invalidIntent(nil)
		}
		e, err := expectedRecord("attempt", nonce, prior)
		if err != nil {
			return nil, err
		}
		expected.Attempts[nonce] = attemptChange{Expect: e, Value: changed}
	case "complete":
		prior := old.Attempts[nonce]
		if prior == nil || prior.Phase != "response_ready" {
			return nil, invalidIntent(nil)
		}
		view := selectRefreshState(old, prior, nil, nonce)
		g, b, a := selectedKeys(view)
		if err := view.complete(nonce, view.Attempts[nonce]); err != nil {
			return nil, invalidIntent(err)
		}
		return selectedChanges(old, view, g, b, a)
	case "uncertain":
		prior := old.Attempts[nonce]
		if prior == nil || (prior.Phase != "prepared" && prior.Phase != "response_ready") {
			return nil, invalidIntent(nil)
		}
		copy := *prior
		copy.Phase = "uncertain"
		e, err := expectedRecord("attempt", nonce, prior)
		if err != nil {
			return nil, err
		}
		expected.Attempts[nonce] = attemptChange{Expect: e, Value: &copy}
	case "retire_bindings":
		if d.Attempt != nil || len(d.Retirements) == 0 {
			return nil, invalidIntent(nil)
		}
		view := selectRefreshState(old, nil, d.Retirements, "")
		g, b, a := selectedKeys(view)
		used, err := view.retireCapturedSet(d.Retirements)
		if err != nil || len(used) != len(d.Retirements) {
			return nil, invalidIntent(err)
		}
		return selectedChanges(old, view, g, b, a)
	default:
		return nil, invalidIntent(nil)
	}
	return expected, nil
}

func validateRefreshDelta(old *refreshJournal, d *refreshDelta) error {
	if d == nil || d.Generations == nil || d.Bindings == nil || d.Attempts == nil || len(d.Generations)+len(d.Bindings)+len(d.Attempts) == 0 {
		return invalidIntent(nil)
	}
	if d.Transition == "retire_bindings" {
		if d.Attempt != nil || len(d.Retirements) == 0 {
			return invalidIntent(nil)
		}
	} else if d.Attempt == nil || !digestValid(*d.Attempt) || len(d.Retirements) != 0 {
		return invalidIntent(nil)
	}
	for k, c := range d.Generations {
		if !digestValid(k) || c.Value == nil || !expectationMatches("generation", k, old.Generations[k], c.Expect) || reflect.DeepEqual(old.Generations[k], c.Value) {
			return invalidIntent(nil)
		}
		if old.Generations[k] != nil && old.Generations[k].Consumed && !c.Value.Consumed {
			return invalidIntent(nil)
		}
	}
	for k, c := range d.Bindings {
		if !digestValid(k) || !expectationMatches("binding", k, old.Bindings[k], c.Expect) || reflect.DeepEqual(old.Bindings[k], c.Value) {
			return invalidIntent(nil)
		}
	}
	for k, c := range d.Attempts {
		if !validAttempt(k, c.Value) || !expectationMatches("attempt", k, old.Attempts[k], c.Expect) || reflect.DeepEqual(old.Attempts[k], c.Value) {
			return invalidIntent(nil)
		}
	}
	expected, err := expectedRefreshChanges(old, d)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected.Generations, d.Generations) || !reflect.DeepEqual(expected.Bindings, d.Bindings) || !reflect.DeepEqual(expected.Attempts, d.Attempts) {
		return invalidIntent(nil)
	}
	getAttempt := func(k string) *refreshAttempt {
		if c, ok := d.Attempts[k]; ok {
			return c.Value
		}
		return old.Attempts[k]
	}
	getGeneration := func(k string) *refreshGeneration {
		if c, ok := d.Generations[k]; ok {
			return c.Value
		}
		return old.Generations[k]
	}
	for g, c := range d.Generations {
		v := c.Value
		if !optionalValid(v.ActiveEpoch) || !optionalValid(v.Reservation) || (v.ActiveEpoch != nil && v.Reservation != nil) {
			return invalidIntent(nil)
		}
		if v.ActiveEpoch != nil {
			a := getAttempt(*v.ActiveEpoch)
			if a == nil || a.Phase != "completed" || deref(a.ResultGeneration) != g {
				return invalidIntent(nil)
			}
		}
		if v.Reservation != nil {
			a := getAttempt(*v.Reservation)
			if !v.Consumed || a == nil || a.Kind != "refresh" || deref(a.InputGeneration) != g || a.Phase == "completed" {
				return invalidIntent(nil)
			}
		}
	}
	indexes := map[string]map[string]refreshWrite{}
	for k, c := range d.Bindings {
		b := c.Value
		if b == nil {
			continue
		}
		if !digestValid(b.Generation) || !digestValid(b.Epoch) || !digestValid(b.Entry) || getGeneration(b.Generation) == nil {
			return invalidIntent(nil)
		}
		a := getAttempt(b.Epoch)
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
		w, ok := index[k]
		if !ok || deref(w.AfterGeneration) != b.Generation || deref(w.AfterEpoch) != b.Epoch || w.AfterEntry != b.Entry {
			return invalidIntent(nil)
		}
	}
	return nil
}

package dstate

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/deployplan"
)

// WithBindings returns an independent snapshot for previewing deployment after binding.
func (db *DeploymentState) WithBindings(bindings map[string]*deployplan.BindEntry) Database {
	db.AssertOpenedForRead()
	db.mu.Lock()
	defer db.mu.Unlock()
	data := db.Data
	data.Features = maps.Clone(data.Features)
	data.State = make(map[string]ResourceEntry, len(db.Data.State)+len(bindings))
	maps.Copy(data.State, db.Data.State)
	for key, entry := range bindings {
		data.State[key] = ResourceEntry{ID: entry.ID, State: entry.State, DependsOn: entry.DependsOn}
	}
	return data
}

// Bind records bindings without changing the workspace resources.
func (db *DeploymentState) Bind(ctx context.Context, bindings map[string]*deployplan.BindEntry) error {
	db.AssertOpenedForWrite()
	keys := slices.Sorted(maps.Keys(bindings))
	for _, key := range keys {
		if id := db.GetResourceID(key); id != "" {
			return fmt.Errorf("cannot bind %s: already bound to ID %q", key, id)
		}
		if bindings[key].ID == "" {
			return fmt.Errorf("cannot bind %s: missing resource ID", key)
		}
	}
	for _, key := range keys {
		entry := bindings[key]
		if err := db.SaveState(ctx, key, entry.ID, entry.State, entry.DependsOn); err != nil {
			return fmt.Errorf("binding %s: %w", key, err)
		}
	}
	return nil
}

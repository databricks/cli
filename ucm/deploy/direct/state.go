package direct

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// StateVersion is bumped on incompatible changes to the on-disk state shape.
// Direct-engine state is strictly a local cache — there is no remote mirror,
// unlike terraform state — but the versioning discipline is retained so that
// an older CLI refuses to read a newer format.
const StateVersion = 1

// StateFileName is the per-target file where direct-engine recorded state is
// persisted. Sits next to (but independent of) the terraform-engine artifacts
// under `.databricks/ucm/<target>/`.
const StateFileName = "resources.json"

// State is the recorded snapshot of every resource the direct engine has
// successfully applied. Keys match the plan's per-resource keys so the
// next plan can diff desired vs recorded by a simple map lookup.
type State struct {
	Version  int                      `json:"version"`
	Catalogs map[string]*CatalogState `json:"catalogs,omitempty"`
	Schemas  map[string]*SchemaState  `json:"schemas,omitempty"`
	Grants   map[string]*GrantState   `json:"grants,omitempty"`
}

// CatalogState is what the direct engine records for a catalog after a
// successful apply. Shape mirrors the slice of fields the UCM catalog
// resource currently models — expand only as the resource model grows.
type CatalogState struct {
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	StorageRoot string            `json:"storage_root,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// SchemaState mirrors CatalogState's discipline for the schema resource.
type SchemaState struct {
	Name    string            `json:"name"`
	Catalog string            `json:"catalog"`
	Comment string            `json:"comment,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// GrantState records a single grant's inputs. Grants are keyed in UCM config
// by arbitrary user-chosen keys; the (securable, principal, privileges) triple
// is the semantic identity we compare when planning.
type GrantState struct {
	SecurableType string   `json:"securable_type"`
	SecurableName string   `json:"securable_name"`
	Principal     string   `json:"principal"`
	Privileges    []string `json:"privileges"`
}

// NewState returns an empty State ready to be populated by the planner.
func NewState() *State {
	return &State{
		Version:  StateVersion,
		Catalogs: make(map[string]*CatalogState),
		Schemas:  make(map[string]*SchemaState),
		Grants:   make(map[string]*GrantState),
	}
}

// LoadState reads the recorded direct-engine state from the given path.
// Returns (NewState(), nil) when the file is absent — first-run is not an
// error for the direct engine (unlike terraform which requires init first).
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("read direct state %s: %w", path, err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse direct state %s: %w", path, err)
	}
	if s.Version > StateVersion {
		return nil, fmt.Errorf("direct state %s: version %d > supported %d; upgrade the CLI", path, s.Version, StateVersion)
	}
	if s.Catalogs == nil {
		s.Catalogs = make(map[string]*CatalogState)
	}
	if s.Schemas == nil {
		s.Schemas = make(map[string]*SchemaState)
	}
	if s.Grants == nil {
		s.Grants = make(map[string]*GrantState)
	}
	return &s, nil
}

// SaveState writes state atomically (write to a sibling tmp file, then rename)
// so a crash mid-write never leaves behind a truncated blob.
func SaveState(path string, s *State) error {
	if s.Version == 0 {
		s.Version = StateVersion
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal direct state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write direct state tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace direct state %s: %w", path, err)
	}
	return nil
}

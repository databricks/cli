package config

type Experimental struct {
	Scripts map[ScriptHook]Command `json:"scripts,omitempty"`
}

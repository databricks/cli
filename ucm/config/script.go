package config

// Script is a user-defined shell command bound to a phase hook.
// Mirrors bundle.config.Script.
type Script struct {
	// Content of the script to be executed.
	Content string `json:"content"`
}

// ScriptHook names a phase boundary at which a Script may be invoked.
// UCM's hook set is narrower than bundle's: only the phases that map to
// user-visible operations are exposed. Terraform-render is omitted on
// purpose — it has no user-facing pre/post equivalent in UCM.
type ScriptHook = string

const (
	ScriptPreInit     ScriptHook = "pre_init"
	ScriptPostInit    ScriptHook = "post_init"
	ScriptPreDeploy   ScriptHook = "pre_deploy"
	ScriptPostDeploy  ScriptHook = "post_deploy"
	ScriptPreDestroy  ScriptHook = "pre_destroy"
	ScriptPostDestroy ScriptHook = "post_destroy"
)

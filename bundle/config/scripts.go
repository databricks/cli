package config

type Command string
type ScriptHook string

const (
	ScriptPreBuild   ScriptHook = "prebuild"
	ScriptPostBuild  ScriptHook = "postbuild"
	ScriptPreDeploy  ScriptHook = "predeploy"
	ScriptPostDeploy ScriptHook = "postdeploy"
)

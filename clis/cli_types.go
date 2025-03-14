package clis

// CLIType represents the type of CLI being used
type CLIType int

const (
	// General is the standard CLI with all commands
	General CLIType = iota

	// DLT is the CLI focused on DLT/bundle functionality
	DLT

	// DABs is the CLI focused only on bundle functionality
	DAB
)

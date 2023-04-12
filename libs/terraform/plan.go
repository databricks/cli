package terraform

type Plan struct {
	// Path to the plan
	Path string

	// Holds whether the user can consented to destruction. Either by interactive
	// confirmation or by passing a command line flag
	ConfirmApply bool

	// If true, the plan is empty and applying it will not do anything
	IsEmpty bool
}

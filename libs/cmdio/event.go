package cmdio

type Event interface {
	// convert event into human readable string
	String() string

	// true if event supports inplace logging, return false otherwise
	IsInplaceSupported() bool
}

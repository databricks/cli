package cmdio

type Event interface {
	// convert event into human readable string
	String() string
}

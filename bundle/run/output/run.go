package output

type RunOutput interface {
	String() (string, error)
}

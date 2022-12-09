package jsonflag

type jsonFlag struct {
	raw string
}

func (j *jsonFlag) String() string
func (j *jsonFlag) Set(string) error
func (j *jsonFlag) Type() string

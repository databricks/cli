package bundle

import "context"

type seqMutator struct {
	mutators []Mutator
}

func (s *seqMutator) Name() string {
	return "seq"
}

func (s *seqMutator) Apply(ctx context.Context, b *Bundle) error {
	for _, m := range s.mutators {
		err := Apply(ctx, b, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func Seq(ms ...Mutator) Mutator {
	return &seqMutator{mutators: ms}
}

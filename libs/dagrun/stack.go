package dagrun

type stack[T any] struct{ data []T }

func (s *stack[T]) push(v T) { s.data = append(s.data, v) }
func (s *stack[T]) pop() T   { v := s.data[len(s.data)-1]; s.data = s.data[:len(s.data)-1]; return v }
func (s *stack[T]) peek() *T { return &s.data[len(s.data)-1] }
func (s *stack[T]) len() int { return len(s.data) }

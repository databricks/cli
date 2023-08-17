package set

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	s := NewSetFrom([]string{})
	require.ElementsMatch(t, []string{}, s.Iter())

	s = NewSetFrom([]string{"a", "a", "a", "b", "b", "c", "d", "e"})
	require.ElementsMatch(t, []string{"a", "b", "c", "d", "e"}, s.Iter())

	i := NewSetFrom([]int{1, 1, 2, 3, 4, 5, 7, 7, 7, 10, 11})
	require.ElementsMatch(t, []int{1, 2, 3, 4, 5, 7, 10, 11}, i.Iter())

	f := NewSetFrom([]float32{1.1, 1.1, 2.0, 3.1, 4.5, 5.1, 7.1, 7.2, 7.1, 10.1, 11.0})
	require.ElementsMatch(t, []float32{1.1, 2.0, 3.1, 4.5, 5.1, 7.1, 7.2, 10.1, 11.0}, f.Iter())
}

type testStruct struct {
	key   string
	value int
}

func TestSetCustomKey(t *testing.T) {
	s := NewSetF(func(item *testStruct) string {
		return fmt.Sprintf("%s:%d", item.key, item.value)
	})
	s.Add(&testStruct{"a", 1})
	s.Add(&testStruct{"b", 2})
	s.Add(&testStruct{"c", 1})
	s.Add(&testStruct{"a", 1})
	s.Add(&testStruct{"a", 1})
	s.Add(&testStruct{"a", 1})
	s.Add(&testStruct{"c", 1})
	s.Add(&testStruct{"c", 3})

	require.ElementsMatch(t, []*testStruct{
		{"a", 1},
		{"b", 2},
		{"c", 1},
		{"c", 3},
	}, s.Iter())
}

func TestSetAdd(t *testing.T) {
	s := NewSet[string]()
	s.Add("a")
	s.Add("a")
	s.Add("a")
	s.Add("b")
	s.Add("c")
	s.Add("c")
	s.Add("d")
	s.Add("d")

	require.ElementsMatch(t, []string{"a", "b", "c", "d"}, s.Iter())
}

func TestSetRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("a")
	s.Add("a")
	s.Add("a")
	s.Add("b")
	s.Add("c")
	s.Add("c")
	s.Add("d")
	s.Add("d")

	s.Remove("d")
	s.Remove("d")
	s.Remove("a")

	require.ElementsMatch(t, []string{"b", "c"}, s.Iter())
}

func TestSetHas(t *testing.T) {
	s := NewSet[string]()
	require.False(t, s.Has("a"))

	s.Add("a")
	require.True(t, s.Has("a"))

	s.Add("a")
	s.Add("a")
	require.True(t, s.Has("a"))

	s.Add("b")
	s.Add("c")
	s.Add("c")
	s.Add("d")
	s.Add("d")

	require.True(t, s.Has("a"))
	require.True(t, s.Has("b"))
	require.True(t, s.Has("c"))
	require.True(t, s.Has("d"))

	s.Remove("d")
	s.Remove("a")

	require.False(t, s.Has("a"))
	require.True(t, s.Has("b"))
	require.True(t, s.Has("c"))
	require.False(t, s.Has("d"))
}

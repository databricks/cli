package structtag

import "testing"

func TestJSONTagMethods(t *testing.T) {
	tests := []struct {
		tag           string
		wantName      string
		wantOmitempty bool
		wantOmitzero  bool
	}{
		// empty / degenerate cases
		{"", "", false, false},
		{"-", "-", false, false},

		// name only
		{"id", "id", false, false},

		// leading comma (implicit name = "")
		{",omitempty", "", true, false},

		// single known options
		{"foo,omitzero", "foo", false, true},
		{"bar,omitempty", "bar", true, false},

		// both known options in any order
		{"baz,omitzero,omitempty", "baz", true, true},
		{"baz,omitempty,omitzero", "baz", true, true},

		// unknown options must be ignored
		{"name,string", "name", false, false},
		{"weird,whatever,omitzero,foo", "weird", false, true},
	}

	for _, tt := range tests {
		tag := JSONTag(tt.tag)

		// Test Name method
		if gotName := tag.Name(); gotName != tt.wantName {
			t.Errorf("JSONTag(%q).Name() = %q; want %q", tt.tag, gotName, tt.wantName)
		}

		// Test OmitEmpty method
		if gotOmitEmpty := tag.OmitEmpty(); gotOmitEmpty != tt.wantOmitempty {
			t.Errorf("JSONTag(%q).OmitEmpty() = %v; want %v", tt.tag, gotOmitEmpty, tt.wantOmitempty)
		}

		// Test OmitZero method
		if gotOmitZero := tag.OmitZero(); gotOmitZero != tt.wantOmitzero {
			t.Errorf("JSONTag(%q).OmitZero() = %v; want %v", tt.tag, gotOmitZero, tt.wantOmitzero)
		}
	}
}

var benchTags = []JSONTag{
	"", "-", "id", ",omitempty", "foo,omitzero",
	"baz,omitzero,omitempty", `"q,omitempty"`,
	"name,string", "weird,whatever,omitzero,foo",
}

func BenchmarkJSONTagName(b *testing.B) {
	for range b.N {
		for _, tag := range benchTags {
			tag.Name()
		}
	}
}

func BenchmarkJSONTagOmitEmpty(b *testing.B) {
	for range b.N {
		for _, tag := range benchTags {
			tag.OmitEmpty()
		}
	}
}

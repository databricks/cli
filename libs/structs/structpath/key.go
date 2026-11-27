package structpath

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyString renders p as a resource key: keys joined by dots and indices in brackets,
// without quoting, e.g. "resources.jobs.daily job". State files, plans and deployment
// metadata use this form, so it must not change. Use [ParseKey] to read it back.
func (p *PathNode) KeyString() string {
	var b strings.Builder
	for i, n := range p.AsSlice() {
		if idx, ok := n.Index(); ok {
			fmt.Fprintf(&b, "[%d]", idx)
			continue
		}
		if i > 0 {
			b.WriteByte('.')
		}
		k, _ := n.StringKey()
		b.WriteString(k)
	}
	return b.String()
}

// ParsePathOrKey parses s as a path and, if that fails, as a resource key (see [ParseKey]).
func ParsePathOrKey(s string) (*PathNode, error) {
	p, err := ParsePath(s)
	if err == nil {
		return p, nil
	}
	if p, keyErr := ParseKey(s); keyErr == nil {
		return p, nil
	}
	return nil, err
}

// ParseKey parses a resource key written by [PathNode.KeyString]: keys may hold any
// character except dots and brackets, so a key with spaces or quotes needs no quoting.
func ParseKey(s string) (*PathNode, error) {
	var p *PathNode
	for i, part := range strings.Split(s, ".") {
		key, rest, _ := strings.Cut(part, "[")
		if key == "" && (i > 0 || rest == "") {
			return nil, fmt.Errorf("invalid key %q: empty component", s)
		}
		if key != "" {
			p = NewStringKey(p, key)
		}
		for rest != "" {
			num, after, ok := strings.Cut(rest, "]")
			idx, err := strconv.Atoi(num)
			if !ok || err != nil || idx < 0 {
				return nil, fmt.Errorf("invalid key %q: bad index", s)
			}
			p = NewIndex(p, idx)
			if after == "" {
				break
			}
			if !strings.HasPrefix(after, "[") {
				return nil, fmt.Errorf("invalid key %q: unexpected %q after index", s, after)
			}
			rest = after[1:]
		}
	}
	return p, nil
}

package sqlsafe

import (
	"errors"
	"strings"
	"unicode"
)

type parser struct {
	src []rune
	idx int

	line int
	col  int

	inString       rune
	inLineComment  bool
	inBlockComment bool

	pendingSpace   bool
	lastWriteSpace bool

	builder strings.Builder

	haveKeyword     bool
	firstKeyword    string
	firstKeywordPos Position

	statements    []Statement
	stmtOrigStart int
}

// ParseStatements splits input SQL text into individual statements, removing
// comments and tracking keyword locations.
func ParseStatements(sql string) ([]Statement, error) {
	p := &parser{
		src:           []rune(sql),
		line:          1,
		col:           1,
		stmtOrigStart: 0,
	}
	err := p.parse()
	if err != nil {
		return nil, err
	}
	return p.statements, nil
}

func (p *parser) parse() error {
	for p.idx < len(p.src) {
		r := p.src[p.idx]

		if p.inLineComment {
			if r == '\n' {
				p.inLineComment = false
				p.writeRune(r)
			}
			p.advance(r)
			continue
		}

		if p.inBlockComment {
			if r == '\n' {
				p.writeRune(r)
				p.advance(r)
				continue
			}
			if r == '*' && p.peek() == '/' {
				p.advance(r)
				p.advance('/')
				p.inBlockComment = false
				p.pendingSpace = true
				continue
			}
			p.advance(r)
			continue
		}

		if p.inString != 0 {
			p.writeRune(r)
			if r == p.inString {
				if (r == '\'' || r == '"') && p.peek() == r {
					// Escaped quote within the string literal.
					p.advance(r)
					p.writeRune(r)
				} else {
					p.inString = 0
				}
			} else if r == '\\' && p.inString == '"' && p.peek() != 0 {
				// Handle escaped characters inside double quotes.
				next := p.peek()
				p.advance(r)
				p.writeRune(next)
			}
			p.advance(r)
			continue
		}

		// Handle the start of comments.
		if r == '-' && p.peek() == '-' {
			p.advance(r)
			p.advance('-')
			p.inLineComment = true
			p.pendingSpace = true
			continue
		}
		if r == '/' && p.peek() == '*' {
			p.advance(r)
			p.advance('*')
			p.inBlockComment = true
			p.pendingSpace = true
			continue
		}

		if r == '\'' || r == '"' || r == '`' {
			p.flushPendingSpace()
			p.markKeywordCandidate()
			p.inString = r
			p.writeRune(r)
			p.advance(r)
			continue
		}

		if r == ';' {
			p.finishStatement()
			p.advance(r)
			continue
		}

		if unicode.IsSpace(r) {
			p.writeRune(r)
			p.advance(r)
			continue
		}

		p.flushPendingSpace()
		p.markKeywordCandidate()
		p.writeRune(r)
		p.advance(r)
	}

	if p.inString != 0 {
		return errors.New("unterminated string literal")
	}
	if p.inBlockComment {
		return errors.New("unterminated block comment")
	}

	p.finishStatement()
	return nil
}

func (p *parser) markKeywordCandidate() {
	if p.haveKeyword {
		return
	}
	r := p.src[p.idx]
	if !unicode.IsLetter(r) && r != '_' {
		return
	}
	pos := Position{Line: p.line, Column: p.col}
	var token []rune
	for j := p.idx; j < len(p.src); j++ {
		rr := p.src[j]
		if unicode.IsLetter(rr) || unicode.IsDigit(rr) || rr == '_' {
			token = append(token, unicode.ToUpper(rr))
			continue
		}
		break
	}
	if len(token) == 0 {
		return
	}
	p.haveKeyword = true
	p.firstKeyword = string(token)
	p.firstKeywordPos = pos
}

func (p *parser) finishStatement() {
	text := strings.TrimSpace(p.builder.String())
	if text != "" {
		start := p.stmtOrigStart
		if start < 0 {
			start = 0
		}
		if start > p.idx {
			start = p.idx
		}
		original := strings.TrimSpace(string(p.src[start:p.idx]))
		stmt := Statement{
			Text:            text,
			OriginalText:    original,
			FirstKeyword:    p.firstKeyword,
			FirstKeywordPos: p.firstKeywordPos,
		}
		p.statements = append(p.statements, stmt)
	}
	p.builder.Reset()
	p.haveKeyword = false
	p.firstKeyword = ""
	p.firstKeywordPos = Position{}
	p.pendingSpace = false
	p.lastWriteSpace = false
	p.resetStatementStart()
}

func (p *parser) resetStatementStart() {
	if p.idx < len(p.src) {
		p.stmtOrigStart = p.idx + 1
		return
	}
	p.stmtOrigStart = len(p.src)
}

func (p *parser) writeRune(r rune) {
	p.builder.WriteRune(r)
	p.lastWriteSpace = unicode.IsSpace(r)
}

func (p *parser) flushPendingSpace() {
	if !p.pendingSpace {
		return
	}
	if p.builder.Len() > 0 && !p.lastWriteSpace {
		p.builder.WriteRune(' ')
		p.lastWriteSpace = true
	}
	p.pendingSpace = false
}

func (p *parser) advance(r rune) {
	if r == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	p.idx++
}

func (p *parser) peek() rune {
	if p.idx+1 >= len(p.src) {
		return 0
	}
	return p.src[p.idx+1]
}

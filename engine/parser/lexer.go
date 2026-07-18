package parser

import (
	"unicode"
	"unicode/utf8"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// Lex scans src into a flat slice of tokens terminated by a single zero-width
// [KindEOF] token. Trivia is included, so concatenating every token's Text
// reproduces src byte-for-byte. Unrecognized bytes become [KindError] tokens;
// Lex never panics on malformed input.
func Lex(src string) []Token {
	l := lexer{src: src}
	var toks []Token
	for {
		t := l.next()
		toks = append(toks, t)
		if t.Kind == KindEOF {
			return toks
		}
	}
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) at(off int) byte {
	if j := l.pos + off; j < len(l.src) {
		return l.src[j]
	}
	return 0
}

func (l *lexer) emit(k SyntaxKind, start int) Token {
	return Token{
		Kind:  k,
		Text:  l.src[start:l.pos],
		Range: text.NewRange(text.TextSize(start), text.TextSize(l.pos)),
	}
}

func (l *lexer) next() Token {
	start := l.pos
	if l.pos >= len(l.src) {
		return l.emit(KindEOF, start) // zero-width
	}

	switch c := l.src[l.pos]; {
	case c == ' ' || c == '\t':
		for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
			l.pos++
		}
		return l.emit(KindWhitespace, start)
	case c == '\n':
		l.pos++
		return l.emit(KindNewline, start)
	case c == '\r':
		l.pos++
		if l.pos < len(l.src) && l.src[l.pos] == '\n' {
			l.pos++
		}
		return l.emit(KindNewline, start)
	case c == '/' && l.at(1) == '/':
		l.pos += 2
		for l.pos < len(l.src) && l.src[l.pos] != '\n' && l.src[l.pos] != '\r' {
			l.pos++
		}
		return l.emit(KindLineComment, start)
	case c == '/' && l.at(1) == '*':
		return l.lexBlockComment(start)
	case c == '"':
		return l.lexDelimited(start, '"', KindString)
	case c == '\'':
		return l.lexDelimited(start, '\'', KindQuotedIdent)
	case c >= '0' && c <= '9':
		return l.lexNumber(start)
	}

	// Identifiers may be Unicode, so decode a rune here.
	if r, size := utf8.DecodeRuneInString(l.src[l.pos:]); isIdentStart(r) {
		l.pos += size
		for l.pos < len(l.src) {
			r, size := utf8.DecodeRuneInString(l.src[l.pos:])
			if !isIdentContinue(r) {
				break
			}
			l.pos += size
		}
		return l.emit(KindIdent, start)
	}

	return l.lexPunct(start)
}

func (l *lexer) lexBlockComment(start int) Token {
	l.pos += 2 // consume /*
	for l.pos < len(l.src) {
		if l.src[l.pos] == '*' && l.at(1) == '/' {
			l.pos += 2
			break
		}
		l.pos++
	}
	return l.emit(KindBlockComment, start)
}

// lexDelimited scans a quoted run (string or restricted name) beginning at the
// opening quote. Backslash escapes the next byte. An unterminated run ends at a
// line break or EOF but still emits kind — the lexer stays error-tolerant.
func (l *lexer) lexDelimited(start int, quote byte, kind SyntaxKind) Token {
	l.pos++ // opening quote
	for l.pos < len(l.src) {
		switch l.src[l.pos] {
		case '\\':
			l.pos++
			if l.pos < len(l.src) {
				l.pos++
			}
		case quote:
			l.pos++
			return l.emit(kind, start)
		case '\n', '\r':
			return l.emit(kind, start) // unterminated
		default:
			l.pos++
		}
	}
	return l.emit(kind, start)
}

func (l *lexer) lexNumber(start int) Token {
	kind := KindInt
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.pos++
	}
	// Fraction: a dot only starts a fraction when a digit follows, so "1.." and
	// "1.foo" member access are not mis-lexed as reals.
	if l.at(0) == '.' && isDigit(l.at(1)) {
		kind = KindReal
		l.pos++ // dot
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.pos++
		}
	}
	// Exponent: [eE][+-]?digits.
	if l.at(0) == 'e' || l.at(0) == 'E' {
		j := l.pos + 1
		if j < len(l.src) && (l.src[j] == '+' || l.src[j] == '-') {
			j++
		}
		if j < len(l.src) && isDigit(l.src[j]) {
			kind = KindReal
			l.pos = j + 1
			for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
				l.pos++
			}
		}
	}
	return l.emit(kind, start)
}

func (l *lexer) lexPunct(start int) Token {
	switch l.src[l.pos] {
	case '(':
		l.pos++
		return l.emit(KindLParen, start)
	case ')':
		l.pos++
		return l.emit(KindRParen, start)
	case '{':
		l.pos++
		return l.emit(KindLBrace, start)
	case '}':
		l.pos++
		return l.emit(KindRBrace, start)
	case '[':
		l.pos++
		return l.emit(KindLBracket, start)
	case ']':
		l.pos++
		return l.emit(KindRBracket, start)
	case ';':
		l.pos++
		return l.emit(KindSemicolon, start)
	case ',':
		l.pos++
		return l.emit(KindComma, start)
	case '.':
		if l.at(1) == '.' {
			l.pos += 2
			return l.emit(KindDotDot, start)
		}
		l.pos++
		return l.emit(KindDot, start)
	case ':':
		switch {
		case l.at(1) == '>' && l.at(2) == '>':
			l.pos += 3
			return l.emit(KindRedefines, start)
		case l.at(1) == '>':
			l.pos += 2
			return l.emit(KindSpecializes, start)
		case l.at(1) == ':':
			l.pos += 2
			return l.emit(KindColonColon, start)
		case l.at(1) == '=':
			l.pos += 2
			return l.emit(KindColonEq, start)
		}
		l.pos++
		return l.emit(KindColon, start)
	case '=':
		l.pos++
		return l.emit(KindEq, start)
	case '*':
		l.pos++
		return l.emit(KindStar, start)
	case '+':
		l.pos++
		return l.emit(KindPlus, start)
	case '-':
		if l.at(1) == '>' {
			l.pos += 2
			return l.emit(KindArrow, start)
		}
		l.pos++
		return l.emit(KindMinus, start)
	case '/':
		l.pos++
		return l.emit(KindSlash, start)
	case '~':
		l.pos++
		return l.emit(KindTilde, start)
	case '@':
		l.pos++
		return l.emit(KindAt, start)
	case '#':
		l.pos++
		return l.emit(KindHash, start)
	case '?':
		l.pos++
		return l.emit(KindQuestion, start)
	case '<':
		l.pos++
		return l.emit(KindLt, start)
	case '>':
		l.pos++
		return l.emit(KindGt, start)
	case '&':
		l.pos++
		return l.emit(KindAmp, start)
	case '|':
		l.pos++
		return l.emit(KindPipe, start)
	case '!':
		l.pos++
		return l.emit(KindBang, start)
	case '%':
		l.pos++
		return l.emit(KindPercent, start)
	case '^':
		l.pos++
		return l.emit(KindCaret, start)
	}

	// Unknown byte: emit a single-rune Error token so lexing always progresses.
	_, size := utf8.DecodeRuneInString(l.src[l.pos:])
	if size == 0 {
		size = 1
	}
	l.pos += size
	return l.emit(KindError, start)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }

func isIdentContinue(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

package parser

import "testing"

func TestLexEverySinglePunct(t *testing.T) {
	cases := map[string]SyntaxKind{
		"(": KindLParen, ")": KindRParen, "{": KindLBrace, "}": KindRBrace,
		"[": KindLBracket, "]": KindRBracket, ";": KindSemicolon, ",": KindComma,
		"=": KindEq, "*": KindStar, "+": KindPlus, "/": KindSlash, "%": KindPercent,
		"^": KindCaret, "&": KindAmp, "|": KindPipe, "!": KindBang, "<": KindLt,
		">": KindGt, "?": KindQuestion, "@": KindAt, "#": KindHash, "~": KindTilde,
	}
	for src, want := range cases {
		toks := lex(t, src)
		if len(toks) != 1 || toks[0].Kind != want || toks[0].Text != src {
			t.Errorf("Lex(%q) = %v (%q), want single %v", src, kinds(toks), textOf(toks), want)
		}
	}
}

func textOf(toks []Token) string {
	if len(toks) == 0 {
		return ""
	}
	return toks[0].Text
}

func TestLexNewlineForms(t *testing.T) {
	// \r\n is one Newline token spanning both bytes.
	toks := lex(t, "a\r\nb")
	if !eqKinds(kinds(toks), KindIdent, KindNewline, KindIdent) {
		t.Fatalf("crlf kinds = %v", kinds(toks))
	}
	if toks[1].Text != "\r\n" {
		t.Errorf("crlf newline text = %q, want %q", toks[1].Text, "\r\n")
	}
	// Lone \r is also a Newline.
	toks = lex(t, "a\rb")
	if toks[1].Kind != KindNewline || toks[1].Text != "\r" {
		t.Errorf("lone cr = %v (%q)", toks[1].Kind, toks[1].Text)
	}
}

func TestLexNumberNonExponent(t *testing.T) {
	// "1e" is not a real (no exponent digits): Int then Ident.
	if !eqKinds(kinds(lex(t, "1e")), KindInt, KindIdent) {
		t.Error("1e should be Int Ident")
	}
	// "1e+" — sign but no digit: Int, Ident(e), Plus.
	if !eqKinds(kinds(lex(t, "1e+")), KindInt, KindIdent, KindPlus) {
		t.Error("1e+ should be Int Ident Plus")
	}
	// "1." — trailing dot with no fraction digit: Int then Dot.
	if !eqKinds(kinds(lex(t, "1.")), KindInt, KindDot) {
		t.Error("1. should be Int Dot")
	}
}

func TestLexUnterminatedQuoted(t *testing.T) {
	// Unterminated string ending at newline stays a String token, lossless.
	toks := lex(t, "\"open\nx")
	if toks[0].Kind != KindString || toks[0].Text != "\"open" {
		t.Errorf("unterminated string = %v (%q)", toks[0].Kind, toks[0].Text)
	}
	// Unterminated quoted ident at EOF.
	toks = lex(t, "'open")
	if len(toks) != 1 || toks[0].Kind != KindQuotedIdent || toks[0].Text != "'open" {
		t.Errorf("unterminated quoted ident = %v (%q)", kinds(toks), textOf(toks))
	}
}

func TestLexEmpty(t *testing.T) {
	toks := Lex("")
	if len(toks) != 1 || toks[0].Kind != KindEOF {
		t.Errorf("Lex(\"\") = %v, want [EOF]", kinds(toks))
	}
}

func TestLexSlashNotComment(t *testing.T) {
	// A slash not followed by / or * is division.
	if !eqKinds(kinds(lex(t, "a/b")), KindIdent, KindSlash, KindIdent) {
		t.Error("a/b should be Ident Slash Ident")
	}
}

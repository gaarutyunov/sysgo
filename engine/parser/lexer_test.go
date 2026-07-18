package parser

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine/text"
)

// lex returns the non-EOF tokens; the trailing EOF is checked separately.
func lex(t *testing.T, src string) []Token {
	t.Helper()
	toks := Lex(src)
	if len(toks) == 0 || toks[len(toks)-1].Kind != KindEOF {
		t.Fatalf("Lex(%q) did not end with EOF: %v", src, toks)
	}
	last := toks[len(toks)-1]
	if last.Text != "" || last.Range.Len() != 0 {
		t.Errorf("EOF token not zero-width: %+v", last)
	}
	return toks[:len(toks)-1]
}

// kinds extracts just the kinds for compact assertions.
func kinds(toks []Token) []SyntaxKind {
	out := make([]SyntaxKind, len(toks))
	for i, tk := range toks {
		out[i] = tk.Kind
	}
	return out
}

func eqKinds(a []SyntaxKind, b ...SyntaxKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLexOperatorsLongestMatch(t *testing.T) {
	tests := []struct {
		src  string
		want SyntaxKind
	}{
		{":", KindColon},
		{"::", KindColonColon},
		{":=", KindColonEq},
		{":>", KindSpecializes},
		{":>>", KindRedefines},
		{".", KindDot},
		{"..", KindDotDot},
		{"-", KindMinus},
		{"->", KindArrow},
		{"~", KindTilde},
		{"@", KindAt},
		{"#", KindHash},
		{"{", KindLBrace},
		{";", KindSemicolon},
	}
	for _, tt := range tests {
		toks := lex(t, tt.src)
		if len(toks) != 1 || toks[0].Kind != tt.want {
			t.Errorf("Lex(%q) = %v, want single %v", tt.src, kinds(toks), tt.want)
			continue
		}
		if toks[0].Text != tt.src {
			t.Errorf("Lex(%q) token text = %q, want %q", tt.src, toks[0].Text, tt.src)
		}
	}
}

func TestLexNumbers(t *testing.T) {
	tests := []struct {
		src  string
		want SyntaxKind
	}{
		{"0", KindInt},
		{"42", KindInt},
		{"3.14", KindReal},
		{"10e9", KindReal},
		{"1.5E-3", KindReal},
		{"2e+8", KindReal},
	}
	for _, tt := range tests {
		toks := lex(t, tt.src)
		if len(toks) != 1 || toks[0].Kind != tt.want || toks[0].Text != tt.src {
			t.Errorf("Lex(%q) = %v (text %q), want single %v", tt.src, kinds(toks), toks[0].Text, tt.want)
		}
	}
}

func TestLexIntDotMember(t *testing.T) {
	// "1.foo" is Int, Dot, Ident — the dot is not a decimal point.
	toks := lex(t, "1.foo")
	if !eqKinds(kinds(toks), KindInt, KindDot, KindIdent) {
		t.Errorf("Lex(1.foo) = %v, want Int Dot Ident", kinds(toks))
	}
	// "1..2" is Int DotDot Int (a range).
	toks = lex(t, "1..2")
	if !eqKinds(kinds(toks), KindInt, KindDotDot, KindInt) {
		t.Errorf("Lex(1..2) = %v, want Int DotDot Int", kinds(toks))
	}
}

func TestLexNamesAndLiterals(t *testing.T) {
	toks := lex(t, `part 'quoted name' "a string"`)
	want := []SyntaxKind{KindIdent, KindWhitespace, KindQuotedIdent, KindWhitespace, KindString}
	if !eqKinds(kinds(toks), want...) {
		t.Fatalf("kinds = %v, want %v", kinds(toks), want)
	}
	if toks[2].Text != "'quoted name'" {
		t.Errorf("quoted ident text = %q", toks[2].Text)
	}
	if toks[4].Text != `"a string"` {
		t.Errorf("string text = %q", toks[4].Text)
	}
}

func TestLexStringEscape(t *testing.T) {
	toks := lex(t, `"a\"b"`)
	if len(toks) != 1 || toks[0].Kind != KindString || toks[0].Text != `"a\"b"` {
		t.Errorf("escaped string = %v (%q)", kinds(toks), toks[0].Text)
	}
}

func TestLexComments(t *testing.T) {
	toks := lex(t, "a // line\nb /* block */ c")
	got := kinds(toks)
	want := []SyntaxKind{
		KindIdent, KindWhitespace, KindLineComment, KindNewline,
		KindIdent, KindWhitespace, KindBlockComment, KindWhitespace, KindIdent,
	}
	if !eqKinds(got, want...) {
		t.Errorf("comment kinds = %v, want %v", got, want)
	}
}

func TestLexUnterminated(t *testing.T) {
	// Unterminated block comment runs to EOF but stays lossless.
	toks := lex(t, "/* nope")
	if len(toks) != 1 || toks[0].Kind != KindBlockComment || toks[0].Text != "/* nope" {
		t.Errorf("unterminated block = %v (%q)", kinds(toks), toks[0].Text)
	}
}

func TestLexError(t *testing.T) {
	toks := lex(t, "a $ b")
	// '$' is not recognized -> a single Error token.
	if toks[2].Kind != KindError || toks[2].Text != "$" {
		t.Errorf("error token = %v (%q), want Error $", toks[2].Kind, toks[2].Text)
	}
}

func TestLexUnicodeIdent(t *testing.T) {
	toks := lex(t, "café_λ")
	if len(toks) != 1 || toks[0].Kind != KindIdent || toks[0].Text != "café_λ" {
		t.Errorf("unicode ident = %v (%q)", kinds(toks), toks[0].Text)
	}
}

func TestLexRangesContiguous(t *testing.T) {
	src := "part x : Foo :> Bar;"
	toks := Lex(src)
	var prev text.TextSize
	for i, tk := range toks {
		if tk.Range.Start != prev {
			t.Errorf("token %d (%v) starts at %d, want %d (contiguous)", i, tk.Kind, tk.Range.Start, prev)
		}
		prev = tk.Range.End
	}
	if int(prev) != len(src) {
		t.Errorf("final offset = %d, want %d", prev, len(src))
	}
}

func TestLexLosslessRoundTrip(t *testing.T) {
	src := `package Vehicles {
	// a comment
	part def Vehicle :> Base::Thing {
		attribute mass : Real := 1.5e3;
		attribute 'restricted name' : String = "hi\n";
	}
	action def Drive { in speed : Real; }
}
`
	var b strings.Builder
	for _, tk := range Lex(src) {
		b.WriteString(tk.Text)
	}
	if got := b.String(); got != src {
		t.Errorf("round-trip mismatch:\n got %q\nwant %q", got, src)
	}
}

func TestKindStringAndTrivia(t *testing.T) {
	if KindSpecializes.String() != "Specializes" {
		t.Errorf("String() = %q", KindSpecializes.String())
	}
	if !KindWhitespace.IsTrivia() || !KindLineComment.IsTrivia() || KindIdent.IsTrivia() {
		t.Error("IsTrivia classification wrong")
	}
	// Raw() round-trips through the cst namer.
	if Namer(KindColonColon.Raw()) != "ColonColon" {
		t.Errorf("Namer(Raw) = %q", Namer(KindColonColon.Raw()))
	}
	if SyntaxKind(60000).String() != "SyntaxKind(60000)" {
		t.Errorf("unknown kind String = %q", SyntaxKind(60000).String())
	}
}

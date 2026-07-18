package parser

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
)

// Parse lexes and parses src into a lossless CST and returns the tree. It never
// panics: unexpected input is wrapped in error nodes (KindErrorNode) and the
// result still round-trips to src byte-for-byte.
//
// This slice implements the recursive-descent core plus the top-level skeleton
// (source file → package members, nested bodies, qualified names). The KerML
// and SysML member grammars are added by later slices.
func Parse(src string) *cst.Tree {
	p := &parser{tokens: Lex(src), b: cst.NewBuilder(nil)}
	p.parseSourceFile()
	return p.b.Finish()
}

type parser struct {
	tokens []Token
	pos    int // index into tokens, may point at trivia
	b      *cst.Builder
}

// sigIndex returns the index of the next significant (non-trivia) token at or
// after pos.
func (p *parser) sigIndex() int {
	i := p.pos
	for i < len(p.tokens) && p.tokens[i].Kind.IsTrivia() {
		i++
	}
	return i
}

// current returns the kind of the next significant token, or KindEOF at end.
func (p *parser) current() SyntaxKind {
	return p.tokens[p.sigIndex()].Kind
}

// atKeyword reports whether the next significant token is the identifier kw
// (keywords are contextual — recognized by text, not by a distinct lexer kind).
func (p *parser) atKeyword(kw string) bool {
	t := p.tokens[p.sigIndex()]
	return t.Kind == KindIdent && t.Text == kw
}

// emitTrivia flushes trivia tokens at pos into the current open node, so no
// byte of source is ever dropped.
func (p *parser) emitTrivia() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Kind.IsTrivia() {
		t := p.tokens[p.pos]
		p.b.Token(t.Kind.Raw(), t.Text)
		p.pos++
	}
}

// bump emits leading trivia and then the next significant token into the
// current node, advancing past it. The zero-width EOF token is never emitted.
func (p *parser) bump() {
	p.emitTrivia()
	if p.pos < len(p.tokens) && p.tokens[p.pos].Kind != KindEOF {
		t := p.tokens[p.pos]
		p.b.Token(t.Kind.Raw(), t.Text)
		p.pos++
	}
}

func (p *parser) parseSourceFile() {
	p.b.StartNode(KindSourceFile.Raw())
	for p.current() != KindEOF {
		p.parseMember()
	}
	p.emitTrivia() // trailing trivia before EOF stays in the tree
	p.b.FinishNode()
}

func (p *parser) parseMember() {
	if p.atKeyword("package") {
		p.parsePackage()
		return
	}
	p.parseErrorMember()
}

func (p *parser) parsePackage() {
	p.b.StartNode(KindPackage.Raw())
	p.bump() // 'package'
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	switch p.current() {
	case KindLBrace:
		p.parseBody()
	case KindSemicolon:
		p.bump()
	}
	p.b.FinishNode()
}

func (p *parser) parseBody() {
	p.b.StartNode(KindBody.Raw())
	p.bump() // '{'
	for {
		switch p.current() {
		case KindRBrace:
			p.bump() // '}'
			p.b.FinishNode()
			return
		case KindEOF:
			p.b.FinishNode() // unterminated body — tolerant, still lossless
			return
		default:
			p.parseMember()
		}
	}
}

func (p *parser) parseQualifiedName() {
	p.b.StartNode(KindQualifiedName.Raw())
	p.parseNameSegment()
	for p.current() == KindColonColon {
		p.bump() // '::'
		p.parseNameSegment()
	}
	p.b.FinishNode()
}

func (p *parser) parseNameSegment() {
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.b.StartNode(KindName.Raw())
		p.bump()
		p.b.FinishNode()
		return
	}
	p.parseErrorToken() // '::' not followed by a name
}

// parseErrorMember consumes an unrecognized run into an error node, recovering
// at the next member boundary (';', '}', EOF, or a 'package' keyword).
func (p *parser) parseErrorMember() {
	p.b.StartNode(KindErrorNode.Raw())
	p.bump() // at least one token, so progress is guaranteed
	for {
		switch {
		case p.current() == KindEOF, p.current() == KindRBrace, p.atKeyword("package"):
			p.b.FinishNode()
			return
		case p.current() == KindSemicolon:
			p.bump() // fold the terminating ';' into the error
			p.b.FinishNode()
			return
		default:
			p.bump()
		}
	}
}

// parseErrorToken wraps a single unexpected significant token in an error node.
func (p *parser) parseErrorToken() {
	p.b.StartNode(KindErrorNode.Raw())
	if p.current() != KindEOF {
		p.bump()
	}
	p.b.FinishNode()
}

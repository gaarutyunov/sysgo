package parser

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
)

// Parse lexes and parses src into a lossless CST and returns the tree. It never
// panics: unexpected input is wrapped in error nodes (KindErrorNode) and the
// result still round-trips to src byte-for-byte.
//
// The grammar covers the top-level skeleton (source file → members, nested
// bodies, qualified names) plus the KerML declaration layer: visibility
// prefixes, imports (with ::* / ::** wildcards), type/classifier/feature
// declarations, relationship clauses (specialization, subsetting, redefinition,
// feature typing, conjugation), and feature values. Full semantic resolution is
// a later engine layer; here relationships are only shaped into CST nodes.
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

// sigTokenN returns the n-th significant (non-trivia) token at or after pos, or
// the trailing EOF token when fewer than n+1 remain.
func (p *parser) sigTokenN(n int) Token {
	count := 0
	for i := p.pos; i < len(p.tokens); i++ {
		if p.tokens[i].Kind.IsTrivia() {
			continue
		}
		if count == n {
			return p.tokens[i]
		}
		count++
	}
	return p.tokens[len(p.tokens)-1] // EOF
}

// current returns the kind of the next significant token, or KindEOF at end.
func (p *parser) current() SyntaxKind { return p.sigTokenN(0).Kind }

// atKeyword reports whether the next significant token is the identifier kw
// (keywords are contextual — recognized by text, not by a distinct lexer kind).
func (p *parser) atKeyword(kw string) bool {
	t := p.sigTokenN(0)
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

// parseMember dispatches on the member keyword, looking past an optional
// visibility prefix to decide the member kind.
func (p *parser) parseMember() {
	vis := p.atVisibility()
	kwIdx := 0
	if vis {
		kwIdx = 1
	}
	kw := ""
	if t := p.sigTokenN(kwIdx); t.Kind == KindIdent {
		kw = t.Text
	}

	switch {
	case kw == "import":
		p.parseImport(vis)
	case kw == "package":
		p.parsePackage(vis)
	case kw == "abstract" || isTypeKeyword(kw):
		p.parseTypeDecl(vis)
	default:
		// Not a recognized member (possibly a stray visibility keyword) — the
		// error recovery swallows it, visibility and all.
		p.parseErrorMember()
	}
}

func (p *parser) atVisibility() bool {
	t := p.sigTokenN(0)
	return t.Kind == KindIdent && (t.Text == "public" || t.Text == "private" || t.Text == "protected")
}

func isTypeKeyword(s string) bool {
	switch s {
	case "namespace", "type", "classifier", "class", "struct", "datatype",
		"feature", "behavior", "function":
		return true
	default:
		return false
	}
}

// bumpVisibility emits the leading visibility keyword wrapped in a Visibility
// node.
func (p *parser) bumpVisibility() {
	p.b.StartNode(KindVisibility.Raw())
	p.bump()
	p.b.FinishNode()
}

func (p *parser) parsePackage(vis bool) {
	p.b.StartNode(KindPackage.Raw())
	if vis {
		p.bumpVisibility()
	}
	p.bump() // 'package'
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	p.parseBodyOrSemi()
	p.b.FinishNode()
}

func (p *parser) parseTypeDecl(vis bool) {
	p.b.StartNode(KindTypeDecl.Raw())
	if vis {
		p.bumpVisibility()
	}
	if p.atKeyword("abstract") {
		p.bump() // 'abstract' modifier
	}
	p.bump() // the type keyword
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	p.parseRelationships()
	p.parseFeatureValueOpt()
	p.parseBodyOrSemi()
	p.b.FinishNode()
}

func (p *parser) parseImport(vis bool) {
	p.b.StartNode(KindImport.Raw())
	if vis {
		p.bumpVisibility()
	}
	p.bump() // 'import'
	p.parseImportName()
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseImportName parses a qualified name optionally ending in a ::* (all
// members) or ::** (recursive) wildcard.
func (p *parser) parseImportName() {
	p.b.StartNode(KindImportName.Raw())
	switch p.current() {
	case KindIdent, KindQuotedIdent:
		p.parseNameSegment()
		for p.current() == KindColonColon {
			p.bump() // '::'
			switch p.current() {
			case KindIdent, KindQuotedIdent:
				p.parseNameSegment()
			case KindStar:
				p.bump() // '*'
				if p.current() == KindStar {
					p.bump() // recursive '**'
				}
				p.b.FinishNode()
				return
			default:
				p.b.FinishNode()
				return
			}
		}
	case KindStar:
		p.bump()
		if p.current() == KindStar {
			p.bump()
		}
	}
	p.b.FinishNode()
}

// parseRelationships consumes zero or more relationship clauses. A clause is a
// relationship operator (:>, :>>, :, ~) or keyword (specializes, subsets,
// redefines, conjugates) followed by a comma-separated list of target names.
func (p *parser) parseRelationships() {
	for p.atRelationship() {
		p.b.StartNode(KindRelationship.Raw())
		p.bump() // operator or keyword
		p.parseQualifiedName()
		for p.current() == KindComma {
			p.bump()
			p.parseQualifiedName()
		}
		p.b.FinishNode()
	}
}

func (p *parser) atRelationship() bool {
	switch p.current() {
	case KindSpecializes, KindRedefines, KindColon, KindTilde:
		return true
	}
	return p.atKeyword("specializes") || p.atKeyword("subsets") ||
		p.atKeyword("redefines") || p.atKeyword("conjugates")
}

func (p *parser) parseFeatureValueOpt() {
	if c := p.current(); c == KindEq || c == KindColonEq {
		p.b.StartNode(KindFeatureValue.Raw())
		p.bump() // '=' or ':='
		p.parseExpr()
		p.b.FinishNode()
	}
}

// parseExpr parses a primary expression: a name reference or a literal. Richer
// expression grammar is deferred to a later slice.
func (p *parser) parseExpr() {
	p.b.StartNode(KindExpr.Raw())
	switch p.current() {
	case KindIdent, KindQuotedIdent:
		p.parseQualifiedName()
	case KindInt, KindReal, KindString:
		p.bump()
	}
	p.b.FinishNode()
}

func (p *parser) parseBodyOrSemi() {
	switch p.current() {
	case KindLBrace:
		p.parseBody()
	case KindSemicolon:
		p.bump()
	}
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
	p.parseErrorToken() // e.g. '::' not followed by a name
}

// parseErrorMember consumes an unrecognized run into an error node, recovering
// at the next member boundary (';', '}', EOF, or a member-start keyword).
func (p *parser) parseErrorMember() {
	p.b.StartNode(KindErrorNode.Raw())
	p.bump() // at least one token, so progress is guaranteed
	for {
		switch {
		case p.current() == KindEOF, p.current() == KindRBrace, p.atMemberStart():
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

// atMemberStart reports whether the next significant token begins a member,
// used as an error-recovery synchronization point.
func (p *parser) atMemberStart() bool {
	if p.atVisibility() {
		return true
	}
	if p.atKeyword("import") || p.atKeyword("package") || p.atKeyword("abstract") {
		return true
	}
	t := p.sigTokenN(0)
	return t.Kind == KindIdent && isTypeKeyword(t.Text)
}

// parseErrorToken wraps a single unexpected significant token in an error node.
func (p *parser) parseErrorToken() {
	p.b.StartNode(KindErrorNode.Raw())
	if p.current() != KindEOF {
		p.bump()
	}
	p.b.FinishNode()
}

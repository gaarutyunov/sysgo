package parser

import (
	"github.com/gaarutyunov/sysgo/engine/cst"
)

// Parse lexes and parses src into a lossless CST and returns the tree. It never
// panics: unexpected input is wrapped in error nodes (KindErrorNode) and the
// result still round-trips to src byte-for-byte.
//
// The grammar covers the top-level skeleton (source file → members, nested
// bodies, qualified names), the KerML declaration layer (visibility, imports
// with ::* / ::** wildcards, type/classifier/feature declarations, relationship
// clauses, feature values), and the SysML layer (part/attribute/item/action/
// port/... definitions and usages, direction/ref modifiers, multiplicity, and
// @/# prefix annotations). Full semantic resolution is a later engine layer;
// here everything is only shaped into CST nodes.
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

// parseMember dispatches on the member keyword, looking past optional @/#
// annotations and a visibility prefix to decide the member kind.
func (p *parser) parseMember() {
	switch kw := p.lookaheadMemberKeyword(); {
	case kw == "import":
		p.parseImport()
	case kw == "package":
		p.parsePackage()
	case kw == "perform":
		p.parsePerform()
	case kw == "first" || kw == "then" || kw == "succession":
		p.parseSuccession()
	case isControlNodeKeyword(kw):
		p.parseControlNode()
	case kw == "accept":
		p.parseAccept()
	case kw == "transition":
		p.parseTransition()
	case kw == "loop":
		p.parseLoop()
	case isDeclKeyword(kw):
		p.parseDeclaration()
	default:
		p.parseErrorMember()
	}
}

// lookaheadMemberKeyword returns the declaration keyword that starts the next
// member, skipping any leading @/# annotations and a visibility keyword. It
// returns "" when no identifier keyword is found.
func (p *parser) lookaheadMemberKeyword() string {
	idx := 0
	for {
		t := p.sigTokenN(idx)
		if t.Kind != KindAt && t.Kind != KindHash {
			break
		}
		idx++ // '@' or '#'
		if k := p.sigTokenN(idx).Kind; k == KindIdent || k == KindQuotedIdent {
			idx++
			for p.sigTokenN(idx).Kind == KindColonColon {
				idx++ // '::'
				if k := p.sigTokenN(idx).Kind; k == KindIdent || k == KindQuotedIdent {
					idx++
				}
			}
		}
		// Skip an annotation body { ... } (balanced braces) so the member
		// keyword after it is still found.
		if p.sigTokenN(idx).Kind == KindLBrace {
			idx = p.skipBraces(idx)
		}
	}
	if t := p.sigTokenN(idx); t.Kind == KindIdent && isVisibility(t.Text) {
		idx++
	}
	if t := p.sigTokenN(idx); t.Kind == KindIdent {
		return t.Text
	}
	return ""
}

// skipBraces returns the significant-token index just past a balanced { ... }
// group that starts at index start. If braces never balance (malformed input),
// it returns the index at EOF.
func (p *parser) skipBraces(start int) int {
	depth := 0
	for i := start; ; i++ {
		switch p.sigTokenN(i).Kind {
		case KindLBrace:
			depth++
		case KindRBrace:
			depth--
			if depth == 0 {
				return i + 1
			}
		case KindEOF:
			return i
		}
	}
}

func isVisibility(s string) bool {
	return s == "public" || s == "private" || s == "protected"
}

func (p *parser) atVisibility() bool {
	t := p.sigTokenN(0)
	return t.Kind == KindIdent && isVisibility(t.Text)
}

// isTypeKeyword reports the KerML type-declaration keywords.
func isTypeKeyword(s string) bool {
	switch s {
	case "namespace", "type", "classifier", "class", "struct", "datatype",
		"feature", "behavior", "function":
		return true
	default:
		return false
	}
}

// isSysmlKind reports the SysML definition/usage noun keywords.
func isSysmlKind(s string) bool {
	switch s {
	case "part", "attribute", "item", "action", "port", "connection", "interface",
		"metadata", "enum", "constraint", "requirement", "state", "calc",
		"occurrence", "view", "viewpoint", "flow", "allocation", "binding",
		"succession", "rendering", "concern":
		return true
	default:
		return false
	}
}

// isModifier reports the declaration prefix modifiers (direction, ref, etc.).
func isModifier(s string) bool {
	switch s {
	case "abstract", "ref", "in", "out", "inout", "readonly", "derived", "end",
		"composite", "portion", "variation", "individual", "snapshot", "timeslice":
		return true
	default:
		return false
	}
}

func (p *parser) atModifier() bool {
	t := p.sigTokenN(0)
	return t.Kind == KindIdent && isModifier(t.Text)
}

// isDeclKeyword reports whether kw can begin a KerML/SysML declaration.
func isDeclKeyword(s string) bool {
	return isTypeKeyword(s) || isSysmlKind(s) || isModifier(s)
}

// parsePrefix consumes optional @/# annotations and a visibility keyword into
// the current open node.
func (p *parser) parsePrefix() {
	for p.current() == KindAt || p.current() == KindHash {
		p.b.StartNode(KindAnnotation.Raw())
		p.bump() // '@' or '#'
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName()
		}
		// An annotation may carry a body of `name = value;` assignments, e.g.
		// @REST { path = "/orders"; method = POST; }.
		if p.current() == KindLBrace {
			p.parseAnnotationBody()
		}
		p.b.FinishNode()
	}
	if p.atVisibility() {
		p.bumpVisibility()
	}
}

// parseAnnotationBody parses `{ name (: type)? (= value)? ; ... }` — a body of
// bare feature-value assignments carried by a metadata annotation. Bare names
// are not otherwise valid member starts, so this uses a dedicated assignment
// parser rather than parseMember.
func (p *parser) parseAnnotationBody() {
	p.b.StartNode(KindBody.Raw())
	p.bump() // '{'
	for {
		switch p.current() {
		case KindRBrace:
			p.bump() // '}'
			p.b.FinishNode()
			return
		case KindEOF:
			p.b.FinishNode() // unterminated — tolerant
			return
		case KindIdent, KindQuotedIdent:
			p.parseAssignment()
		default:
			p.parseErrorMember() // stray token — recover, guaranteeing progress
		}
	}
}

// parseAssignment parses one `name (: type)? (= value)? ;` assignment as a
// Usage node.
func (p *parser) parseAssignment() {
	cp := p.b.Checkpoint()
	p.parseQualifiedName() // the assigned feature name
	p.parseRelationships() // optional `: Type`
	p.parseFeatureValueOpt()
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.StartNodeAt(cp, KindUsage.Raw())
	p.b.FinishNode()
}

func (p *parser) bumpVisibility() {
	p.b.StartNode(KindVisibility.Raw())
	p.bump()
	p.b.FinishNode()
}

func (p *parser) parsePackage() {
	p.b.StartNode(KindPackage.Raw())
	p.parsePrefix()
	p.bump() // 'package'
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	p.parseBodyOrSemi()
	p.b.FinishNode()
}

func (p *parser) parseImport() {
	p.b.StartNode(KindImport.Raw())
	p.parsePrefix()
	p.bump() // 'import'
	p.parseImportName()
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseDeclaration parses a KerML or SysML declaration and picks the node kind
// once the shape is known: a declaration with 'def' is a Def, one led by a
// KerML type keyword is a TypeDecl, otherwise it is a Usage. The Checkpoint lets
// us defer that choice until after parsing.
func (p *parser) parseDeclaration() {
	cp := p.b.Checkpoint()
	p.parsePrefix()
	for p.atModifier() {
		p.bump() // direction / ref / abstract / ... modifiers
	}
	hadDef := false
	kermlKw := false
	if t := p.sigTokenN(0); t.Kind == KindIdent && (isTypeKeyword(t.Text) || isSysmlKind(t.Text)) {
		kermlKw = isTypeKeyword(t.Text)
		p.bump() // the kind keyword
		if p.atKeyword("def") {
			p.bump()
			hadDef = true
		}
	}
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	p.parseRelationships()
	p.parseMultiplicityOpt()
	p.parseFeatureValueOpt()
	p.parseBodyOrSemi()

	kind := KindUsage
	switch {
	case hadDef:
		kind = KindDef
	case kermlKw:
		kind = KindTypeDecl
	}
	p.b.StartNodeAt(cp, kind.Raw())
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

// parseMultiplicityOpt consumes a [ ... ] multiplicity clause if present.
func (p *parser) parseMultiplicityOpt() {
	if p.current() != KindLBracket {
		return
	}
	p.b.StartNode(KindMultiplicity.Raw())
	p.bump() // '['
	for {
		switch p.current() {
		case KindRBracket:
			p.bump() // ']'
			p.b.FinishNode()
			return
		case KindEOF:
			p.b.FinishNode() // unterminated — tolerant
			return
		default:
			p.bump()
		}
	}
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
	if p.current() == KindAt || p.current() == KindHash || p.atVisibility() {
		return true
	}
	if p.atKeyword("import") || p.atKeyword("package") || p.atKeyword("perform") ||
		p.atKeyword("first") || p.atKeyword("then") || p.atKeyword("succession") ||
		isControlNodeKeyword(p.sigTokenN(0).Text) || p.atKeyword("accept") ||
		p.atKeyword("transition") || p.atKeyword("loop") {
		return true
	}
	t := p.sigTokenN(0)
	return t.Kind == KindIdent && isDeclKeyword(t.Text)
}

// parseAccept parses an accept statement, e.g. `accept sig;`, `accept s : Sig;`,
// `accept after 5;`, `accept at t;`.
func (p *parser) parseAccept() {
	p.b.StartNode(KindAccept.Raw())
	p.bump() // 'accept'
	if p.atKeyword("after") || p.atKeyword("at") {
		p.bump() // 'after' / 'at'
	}
	switch p.current() {
	case KindIdent, KindQuotedIdent:
		p.parseQualifiedName() // signal reference or duration/time reference
	case KindInt, KindReal, KindString:
		p.b.StartNode(KindExpr.Raw())
		p.bump() // a literal duration / time value
		p.b.FinishNode()
	}
	p.parseRelationships() // optional `: SignalType`
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseTransition parses a state transition inside a `state def` body, e.g.
//
//	transition first S1 then S2;
//	transition T1 first S1 accept sig if guard do effect then S2;
//
// The optional leading name, `first` source, `accept` trigger, `if` guard,
// `do` effect, and `then` target are each optional except that a well-formed
// transition names a target. Parsing is tolerant: any missing clause is simply
// absent from the node.
func (p *parser) parseTransition() {
	p.b.StartNode(KindTransition.Raw())
	p.bump() // 'transition'
	// Optional transition name: a qualified name before the `first` keyword.
	if c := p.current(); (c == KindIdent || c == KindQuotedIdent) && !p.atKeyword("first") {
		p.parseQualifiedName()
	}
	if p.atKeyword("first") {
		p.bump()
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName() // source state
		}
	}
	if p.atKeyword("accept") {
		p.bump()
		if p.atKeyword("after") || p.atKeyword("at") {
			p.bump()
		}
		switch p.current() {
		case KindIdent, KindQuotedIdent:
			p.parseQualifiedName() // trigger signal reference
		case KindInt, KindReal, KindString:
			p.b.StartNode(KindExpr.Raw())
			p.bump() // literal duration / time value
			p.b.FinishNode()
		}
		p.parseRelationships() // optional `: SignalType` binding on the trigger
	}
	if p.atKeyword("if") {
		p.bump()
		p.parseTransitionGuard()
	}
	if p.atKeyword("do") {
		p.bump()
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName() // effect action reference
		}
	}
	if p.atKeyword("then") {
		p.bump()
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName() // target state
		}
	}
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseTransitionGuard wraps the guard condition — every token between `if` and
// the following `do`/`then`/`;`/`}` — in an Expr node. The guard may be an
// arbitrary boolean expression, so it is captured losslessly as raw tokens
// rather than parsed into structure for this slice.
func (p *parser) parseTransitionGuard() {
	p.b.StartNode(KindExpr.Raw())
	for {
		switch c := p.current(); {
		case c == KindEOF || c == KindRBrace || c == KindSemicolon:
			p.b.FinishNode()
			return
		case p.atKeyword("do") || p.atKeyword("then"):
			p.b.FinishNode()
			return
		default:
			p.bump()
		}
	}
}

// parseLoop parses a repetition, e.g. `loop 3 times Attempt;` or
// `loop retries times Attempt;` — repeat the referenced activity <count> times.
func (p *parser) parseLoop() {
	p.b.StartNode(KindLoop.Raw())
	p.bump() // 'loop'
	switch p.current() {
	case KindInt:
		p.b.StartNode(KindExpr.Raw())
		p.bump() // literal count
		p.b.FinishNode()
	case KindIdent, KindQuotedIdent:
		p.parseQualifiedName() // count reference
	}
	if p.atKeyword("times") {
		p.bump()
	}
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName() // activity reference
	}
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// isControlNodeKeyword reports the control-node keywords.
func isControlNodeKeyword(s string) bool {
	switch s {
	case "fork", "join", "merge", "decide":
		return true
	default:
		return false
	}
}

// parseControlNode parses a control node, e.g. `fork f;` / `join;` / `decide d;`.
func (p *parser) parseControlNode() {
	p.b.StartNode(KindControlNode.Raw())
	p.bump() // fork / join / merge / decide
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName() // optional name
	}
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseSuccession parses a control-flow succession edge, e.g.
//
//	first A then B;   then B;   succession first A then B;
//	first A if guard then B;   (guarded succession)
func (p *parser) parseSuccession() {
	p.b.StartNode(KindSuccession.Raw())
	if p.atKeyword("succession") {
		p.bump()
	}
	if p.atKeyword("first") {
		p.bump()
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName() // source
		}
	}
	if p.atKeyword("if") {
		p.bump()
		p.parseSuccessionGuard()
	}
	if p.atKeyword("then") {
		p.bump()
		if c := p.current(); c == KindIdent || c == KindQuotedIdent {
			p.parseQualifiedName() // target
		}
	}
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseSuccessionGuard wraps the guard condition — every token between `if` and
// the following `then`/`;`/`}` — in an Expr node.
func (p *parser) parseSuccessionGuard() {
	p.b.StartNode(KindExpr.Raw())
	for {
		switch c := p.current(); {
		case c == KindEOF || c == KindRBrace || c == KindSemicolon:
			p.b.FinishNode()
			return
		case p.atKeyword("then"):
			p.b.FinishNode()
			return
		default:
			p.bump()
		}
	}
}

// parsePerform parses a `perform` action reference, e.g.
//
//	perform action charge : ChargeCard;   or   perform ChargeCard;
func (p *parser) parsePerform() {
	p.b.StartNode(KindPerform.Raw())
	p.bump() // 'perform'
	if p.atKeyword("action") {
		p.bump()
	}
	if c := p.current(); c == KindIdent || c == KindQuotedIdent {
		p.parseQualifiedName()
	}
	p.parseRelationships() // optional `: Target`
	if p.current() == KindSemicolon {
		p.bump()
	}
	p.b.FinishNode()
}

// parseErrorToken wraps a single unexpected significant token in an error node.
func (p *parser) parseErrorToken() {
	p.b.StartNode(KindErrorNode.Raw())
	if p.current() != KindEOF {
		p.bump()
	}
	p.b.FinishNode()
}

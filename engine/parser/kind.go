package parser

import (
	"strconv"

	"github.com/gaarutyunov/sysgo/engine/cst"
)

// SyntaxKind is the compact u16 syntax-kind tag for the KerML/SysML grammar. It
// is the explicit raw-tag ↔ typed-kind mapping ENGINE §5 calls for: every kind
// has a stable numeric value, a name, and (for tokens) a trivia classification.
//
// This slice defines the token kinds. Node kinds are appended by the
// recursive-descent core slice; their numeric values continue this enumeration,
// so token and node kinds never collide.
type SyntaxKind uint16

const (
	// KindError is any byte the lexer does not recognize. It keeps lexing
	// error-tolerant — malformed input still yields a lossless token stream.
	KindError SyntaxKind = iota
	// KindEOF is the zero-width terminator token.
	KindEOF

	// Trivia.
	KindWhitespace  // spaces and tabs
	KindNewline     // \n or \r\n
	KindLineComment // // ... to end of line
	KindBlockComment

	// Names and literals.
	KindIdent       // basic identifier; keywords are recognized contextually
	KindQuotedIdent // 'restricted name'
	KindInt         // integer literal
	KindReal        // real literal (fraction and/or exponent)
	KindString      // "double-quoted string"

	// Punctuation and operators.
	KindLParen      // (
	KindRParen      // )
	KindLBrace      // {
	KindRBrace      // }
	KindLBracket    // [
	KindRBracket    // ]
	KindSemicolon   // ;
	KindComma       // ,
	KindDot         // .
	KindDotDot      // ..
	KindColon       // :
	KindColonColon  // ::
	KindColonEq     // :=
	KindSpecializes // :>
	KindRedefines   // :>>
	KindEq          // =
	KindStar        // *
	KindPlus        // +
	KindMinus       // -
	KindSlash       // /
	KindArrow       // ->
	KindTilde       // ~
	KindAt          // @
	KindHash        // #
	KindQuestion    // ?
	KindLt          // <
	KindGt          // >
	KindAmp         // &
	KindPipe        // |
	KindBang        // !
	KindPercent     // %
	KindCaret       // ^
	// Node kinds are appended after KindCaret by the recursive-descent core
	// slice, continuing this iota so token and node kinds never collide.
)

var kindNames = [...]string{
	KindError:        "Error",
	KindEOF:          "EOF",
	KindWhitespace:   "Whitespace",
	KindNewline:      "Newline",
	KindLineComment:  "LineComment",
	KindBlockComment: "BlockComment",
	KindIdent:        "Ident",
	KindQuotedIdent:  "QuotedIdent",
	KindInt:          "Int",
	KindReal:         "Real",
	KindString:       "String",
	KindLParen:       "LParen",
	KindRParen:       "RParen",
	KindLBrace:       "LBrace",
	KindRBrace:       "RBrace",
	KindLBracket:     "LBracket",
	KindRBracket:     "RBracket",
	KindSemicolon:    "Semicolon",
	KindComma:        "Comma",
	KindDot:          "Dot",
	KindDotDot:       "DotDot",
	KindColon:        "Colon",
	KindColonColon:   "ColonColon",
	KindColonEq:      "ColonEq",
	KindSpecializes:  "Specializes",
	KindRedefines:    "Redefines",
	KindEq:           "Eq",
	KindStar:         "Star",
	KindPlus:         "Plus",
	KindMinus:        "Minus",
	KindSlash:        "Slash",
	KindArrow:        "Arrow",
	KindTilde:        "Tilde",
	KindAt:           "At",
	KindHash:         "Hash",
	KindQuestion:     "Question",
	KindLt:           "Lt",
	KindGt:           "Gt",
	KindAmp:          "Amp",
	KindPipe:         "Pipe",
	KindBang:         "Bang",
	KindPercent:      "Percent",
	KindCaret:        "Caret",
}

// String returns the kind's name, or "SyntaxKind(N)" for an unknown value.
func (k SyntaxKind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "SyntaxKind(" + strconv.Itoa(int(k)) + ")"
}

// IsTrivia reports whether the kind is whitespace, a newline, or a comment —
// tokens the parser attaches to structure rather than treating as syntax.
func (k SyntaxKind) IsTrivia() bool {
	switch k {
	case KindWhitespace, KindNewline, KindLineComment, KindBlockComment:
		return true
	default:
		return false
	}
}

// Raw converts the kind to a cst.RawKind for storage in the green tree.
func (k SyntaxKind) Raw() cst.RawKind { return cst.RawKind(k) }

// Namer is a cst.KindNamer that renders green-tree RawKinds produced by this
// package.
func Namer(k cst.RawKind) string { return SyntaxKind(k).String() }

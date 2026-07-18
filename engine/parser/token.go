package parser

import "github.com/gaarutyunov/sysgo/engine/text"

// Token is a single lexical token: its kind, the exact source text it covers
// (including quotes/trivia so the stream stays lossless), and its absolute byte
// range in the source.
type Token struct {
	Kind  SyntaxKind
	Text  string
	Range text.TextRange
}

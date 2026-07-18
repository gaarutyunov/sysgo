// Package parser is the front end of the sysgo engine (specs/ENGINE.md §5b): a
// hand-written recursive-descent parser that turns KerML/SysML source into the
// lossless CST from package cst.
//
// This file set implements the first slice — the lexer. [Lex] scans source into
// a flat slice of [Token]s covering the KerML/SysML lexical grammar. Trivia
// (whitespace, newlines, comments) is emitted as ordinary tokens so the token
// stream is lossless: concatenating every token's text reproduces the source
// byte-for-byte. Any unrecognized byte becomes a single [KindError] token, so
// the lexer never fails on malformed input.
//
// Keywords are contextual: alphanumeric words are lexed as [KindIdent] and the
// parser recognizes keywords by their text. This keeps the lexer stable against
// the large, evolving KerML/SysML keyword set.
//
// [SyntaxKind] is the compact u16 tag shared by tokens (and, in later slices,
// nodes); it converts to cst.RawKind, which is the explicit raw-tag ↔
// typed-kind mapping ENGINE §5 calls for.
package parser

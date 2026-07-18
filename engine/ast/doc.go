// Package ast is the typed syntax view of the sysgo engine (specs/ENGINE.md,
// the syntax layer). It layers strongly-typed accessors over the untyped
// red/green CST from package cst, mirroring rust-analyzer's AstNode pattern:
//
//   - Every typed node ([SourceFile], [Package], [Import], [Declaration],
//     [QualifiedName], …) is a thin wrapper around a cst.Node. Wrappers own no
//     data; they are cheap to create and re-derive everything on demand, so the
//     typed view stays in sync with the lossless tree underneath.
//   - Cast functions (As…) return (T, ok) after checking the node's kind, and
//     accessors return child nodes already wrapped in their typed form.
//   - [Inspect] walks the tree like go/ast.Inspect.
//   - [Format] reproduces source from the AST. Because the CST is lossless, the
//     formatter is faithful: formatting a parsed tree yields the original bytes
//     exactly. Canonical (re-indenting) formatting is a later addition.
//
// The typed layer reads the syntax kinds produced by package parser, so a node
// built by parser.Parse can be viewed directly.
package ast

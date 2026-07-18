// Package cst is Layer 1 of the sysgo engine (specs/ENGINE.md §5): a lossless
// red/green concrete syntax tree.
//
// The design mirrors rowan/cstree:
//
//   - Green nodes are immutable, interned, and deduplicated. Identical subtrees
//     are stored once (hash-consing). Green data carries no absolute position,
//     only a kind, children, and — for tokens — the source text.
//   - Green storage is arena/index-based. Nodes and tokens live in flat slices
//     on the [Tree]; children are referenced by integer index ([childRef]), not
//     by pointer. This is cache-friendly and keeps GC pressure low on large
//     models (ENGINE §5, E11).
//   - The red tree is computed on demand as lightweight cursors ([Node],
//     [Token]) — a green index plus an absolute text offset plus a parent
//     cursor. Red cursors are not persisted, so they add no permanent memory
//     and reintroduce no pointer graph.
//
// The tree is error-tolerant: any node may hold any children, so a parser can
// emit a still-lossless tree for incomplete or incorrect input (ENGINE §5b).
// Kinds are opaque [RawKind] tags here; the concrete KerML/SysML kind set and
// its typed-kind mapping belong to the parser layer (ENGINE §5b).
//
// Losslessness. Concatenating the text of every token in document order
// reproduces the source byte-for-byte, provided the parser feeds trivia
// (whitespace, comments) as tokens. [Node.Text] does exactly that.
//
// Concurrency. A finished [Tree] is immutable and safe for concurrent readers.
// A [Builder] is single-goroutine; build separate trees on separate goroutines
// (they may share one goroutine-safe [text.Interner], ENGINE §5a).
package cst

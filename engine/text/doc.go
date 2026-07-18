// Package text is the bottom foundation layer of the sysgo engine
// (specs/ENGINE.md, Layer 0). It provides the position and identity primitives
// every other engine layer builds on:
//
//   - [Interner] assigns a stable, comparable [Symbol] to each distinct string
//     so downstream layers compare small integer ids instead of bytes.
//   - [FileSet] maps source-file paths to compact [FileId] handles and back.
//   - [TextSize] and [TextRange] describe byte offsets and half-open byte
//     ranges within a source file, mirroring rowan's u32-based model.
//
// Byte, not rune. All offsets and lengths are measured in UTF-8 bytes, so a
// [TextRange] indexes directly into the source string.
//
// Concurrency. [Interner] and [FileSet] are safe for concurrent use: the parser
// interns identifiers and registers files from multiple goroutines
// (ENGINE §5a). [TextSize] and [TextRange] are immutable value types and are
// therefore trivially share-safe.
package text

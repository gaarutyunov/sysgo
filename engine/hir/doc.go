// Package hir is Layer 3 of the sysgo engine (specs/ENGINE.md §7): semantic
// resolution over the typed AST.
//
// This slice builds the symbol layer and name resolution:
//
//   - [Analyze] turns source into a [Model]: a tree of [Symbol]s (namespaces,
//     packages, definitions, usages) with their names, containment, and the
//     imports declared in each scope.
//   - Name resolution ([Model.Resolve]) resolves a simple name by walking
//     outward through enclosing scopes and their imports, and a qualified name
//     segment by segment. Imports support ::* (all members) and ::** (recursive)
//     wildcards and carry their visibility.
//   - Diagnostics (currently: unresolved imports) are produced with source
//     ranges.
//
// The incremental façade [Db] wires the pipeline onto package salsa: source text
// is a durable input and analysis is a tracked query, so editing one file
// recomputes only what changed.
//
// Relationship resolution (specialization/subsetting/redefinition/typing/
// conjugation) and inheritance-chain member lookup are the next slice; full
// stdlib-backed resolution arrives with the project layer.
package hir

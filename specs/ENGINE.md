# SYSGO-ENGINE-SPEC

**Version:** e0.2
**Status:** Architecture and construction fully locked (concurrency, memory
strategy, and parser now decided). Scope of deferred layers noted.
**Parent:** `SYSGO-FRAMEWORK-SPEC.md` (see §2–§5 for platform principles).

---

## 1. Purpose

`engine/` is a **pure-Go SysML v2 + KerML engine** providing full semantic
resolution over a consolidated model. It is the foundation every other layer
consumes: the generators (`gen/`), the future visualizer (`viz/`), and the
downstream consumers codiq and Epos.

The engine is a from-scratch **Go reimplementation that mirrors the architecture
of `syster`** (`github.com/jade-codes/syster-base`, `.../syster-cli`; MIT). It
reproduces syster's design faithfully rather than porting its code, since the
target language differs.

**Fidelity requirement:** full semantic resolution — deep inheritance chains and
resolution against the bundled SysML v2 standard library — not a structural
subset. This is a deliberate choice: generation must be able to resolve
conformant models the way a real tool does.

---

## 2. Reference implementation (syster) and layer map

syster (non-test) is ~36.6k LOC across these modules; the engine maps them to Go
packages as follows:

| syster module | LOC (approx) | Concern | Go package | Status |
| --- | --- | --- | --- | --- |
| `base` | ~0.4k | interning, file ids, text ranges | `engine/text` | in scope |
| `parser` | ~10.7k | lexer + CST + grammar (~207 constructs) | `engine/cst`, `engine/parser` | in scope |
| `syntax` | ~3.6k | typed AST view + formatter | `engine/ast` | in scope |
| `project` | ~0.4k | workspace + stdlib loading | `engine/project` | in scope |
| `hir` | ~8.0k | query DB, symbols, resolve, diagnostics | `engine/salsa`, `engine/hir` | in scope |
| `ide` | ~3.3k | completion, hover, goto | (future) `lsp/` | **deferred** |
| `interchange` | ~10.3k | XMI/YAML/JSON-LD/KPAR + model editing | (future) `viz` export | **deferred (partial)** |

The `ide` layer maps to the deferred language server. The `interchange` layer is
deferred; only the browser-facing JSON export needed by `viz` is anticipated,
and not as part of the engine core (see §8).

---

## 3. Rejected alternatives

Recorded so the pure-Go decision is traceable:

- **cgo / C-ABI static or dynamic lib around syster** — rejected: cgo
  cross-compilation is the primary pain point (per-target C toolchains, no clean
  static cross-builds).
- **Rust→Wasm (syster) via wazero** — rejected: keeps the Go side cgo-free and
  yields one portable artifact, but requires forking syster (its stdlib loader
  uses `rayon` `.par_iter()` at two sites, which fails on single-threaded
  `wasm32-wasip1`) plus an unverified assumption that Salsa 0.18 builds for
  `wasm32-wasip1`. Static analysis only; no successful build was performed.
- **Subprocess (native `syster-cli`) over JSON** — rejected: not true embedding;
  moves cross-compilation to per-platform Rust binaries to vendor/distribute.
- **SysML v2 API & Services (Scala pilot) over REST** — rejected: requires a
  running server + DB as a build dependency.
- **Shelling out to Syside (TypeScript/Langium)** — rejected: adds a
  Node/TypeScript toolchain dependency.
- **Own ANTLR4/tree-sitter grammar from the OMG BNF** — subsumed: this is a
  parser-only path and does not provide the full semantic front end the
  fidelity requirement demands.

The decision to reimplement in Go (rather than embed syster) also reflects the
intent to own the full stack and, later, to build LikeC4-grade tooling and
visualization on the same engine.

---

## 4. Architecture

Four layers, bottom to top:

```
engine/text     interning, FileId, TextSize/TextRange
      |
engine/cst      lossless red/green tree  (Layer 1, §5)
engine/parser   error-tolerant parser producing the CST
engine/ast      typed AST view over the red tree; formatter (lossless round-trip)
      |
engine/salsa    from-scratch incremental query engine  (Layer 2, §6)
engine/hir      symbols, name resolution, KerML semantics, diagnostics (§7)
engine/project  workspace + standard library loading (§8)
      |
Public Go API   in-process typed access to the resolved model  (§9)
```

The incremental design exists to make live tooling (a future LSP and
live-preview visualization) keystroke-fast. Near term, the engine runs mostly in
**batch** behind generation, so the incremental payoff is banked, not yet spent.

**Concurrency model — fine-grained concurrent (§5a).** Queries execute in
parallel with goroutine-safe memo storage and `context` cancellation. Immutable
green nodes are inherently safe to share across goroutines and the GC handles
their lifetime, so no atomic refcounting is needed (unlike cstree's `Send + Sync`
design, which exists only because Rust lacks a GC). The concurrency work
concentrates in the memo store and in concurrent green-node construction (§5),
not in the tree itself.

---

## 5. Layer 1 — Lossless CST (red/green tree)

**Decision: full rowan/cstree-style red/green tree, on an arena/index green
store.**

- Immutable, interned, **deduplicated green nodes** (with pre-hashing of
  subtrees), carrying source text position-independently.
- **Green-node storage: arena / index-based.** Green nodes live in typed arenas
  (slices); children are referenced by integer indices, not pointers. This is
  cache-friendly and minimises GC pressure on models that can reach millions of
  nodes — the reason for choosing it over an idiomatic pointer-graph.
- **Red tree: on-demand cursors, not persisted.** Because a persisted
  pointer-based red tree would reintroduce exactly the GC pressure the arena
  store avoids, the red layer is computed on demand as lightweight cursors
  (green index + text offset + parent info) — which is rowan's actual model.
- Compact kind tags (a `u16`-equivalent), redesigned Go-idiomatically (Go has no
  cheap `transmute`; the raw-tag ↔ typed-kind mapping is explicit).
- Error-tolerant: any node may hold arbitrary children so incomplete/incorrect
  input still yields a tree.

**Rationale:** green-node sharing gives fine-grained incremental reparse (only
changed subtrees rebuilt), low memory on large models, and exact
round-trip/formatting — the properties behind responsive live tooling — and lets
the parser logic mirror syster closely. This is the largest pure-infrastructure
investment and is accepted as such.

**Concurrency implication (from §5a):** under fine-grained concurrent parsing,
multiple files build green nodes in parallel, so the arena append **and** the
hash-consing intern cache must be goroutine-safe — either a synchronized
arena + intern map, or per-goroutine arenas merged at the end. This is the sharp
edge of the concurrency work; the immutable nodes themselves are share-safe for
free under the GC.

**Rejected here:** a pointer-graph GC-managed store (idiomatic but GC-pressure at
scale); an interned-text-only middle path (no fine-grained incremental rebuild);
a plain trivia-preserving tree (no sharing → coarse incremental reparse).

---

## 5b. Parser

**Decision: hand-written recursive-descent, mirroring syster.**

syster's ~10.7k-LOC parser is hand-written across its kerml/sysml grammar
packages; the engine reproduces that. It emits green nodes directly (with trivia
preserved) and provides IDE-grade error recovery — placing error nodes in the
tree while still producing a lossless CST.

**Rationale:** lossless output, trivia preservation, and IDE-grade error recovery
are hard requirements that parser generators do not serve well — the reason
rust-analyzer and syster both hand-write. Full control over error recovery,
direct green-node emission, and near-line-for-line reuse of syster's grammar
logic outweigh the large hand-written surface (~207 constructs) that must be
maintained against SysML/KerML spec revisions.

**Rejected here:** ANTLR4 Go target (builds its own heavy parse tree not green
nodes, hides trivia by default, recovery not lossless-oriented, and is the most
baggage-laden ANTLR runtime — mutex on the hot path, high memory); tree-sitter
(cgo bindings, excluded); parser-combinators (non-idiomatic in Go, discard trivia,
awkward bottom-up green-tree building).

---

## 6. Layer 2 — Incremental query engine (Salsa port)

**Decision: port Salsa's algorithm to Go from scratch.**

- Durable **inputs**, **tracked queries**, revision counters, and automatic
  red-green dependency invalidation that recomputes the minimal set on demand.
- The `parse → symbols → resolve → diagnostics` graph is expressed as queries so
  syster's query definitions can be reproduced almost directly.

**Rationale:** exact Salsa semantics are what keep live tooling fast at scale via
precise automatic invalidation. This is the most novel infrastructure in the
project; without Rust macros the query definitions are more boilerplate-heavy,
and concurrency, memo storage, and cycle handling are subtle — accepted.

**Rejected here:** `go-incr` (general-purpose Jane-Street-incremental analogue;
no first-class durable-input/revision model for language tooling; niche);
gopls-style hand-rolled snapshots + `memoize` + explicit invalidation (manual
invalidation, coarser recomputation, drifts from syster as the query graph
grows).

---

## 7. Layer 3 — Semantic resolution (KerML/SysML)

Full-fidelity semantic layer over the AST:

- Membership and **import** handling (including recursive/wildcard imports and
  visibility).
- **Name resolution** and scoping over qualified names.
- KerML relationships: **specialization, subsetting, redefinition, feature
  typing, conjugation** — resolved along inheritance chains.
- Diagnostics as queries.

Resolution runs against the standard library (§8); most real models specialize
library elements (`Base::`, `Actions::`, `Parts::`, …), so stdlib indexing is
mandatory for correctness, not optional.

---

## 8. Layer 4 — Standard library

The SysML v2 standard library (syster bundles ~94 files / ~25k lines) is
**embedded** in the binary (`go:embed`), parsed, and indexed so name resolution
can resolve against it. It is a required part of the engine, not an external
dependency.

---

## 9. Public API contract

The engine exposes an **in-process, typed Go API over the resolved model**
(per framework §4). No serialized model is part of the core surface.

- Generators, codiq, and Epos import the engine and traverse the resolved model
  directly (type-safe, zero marshalling).
- **Deferred — viz serialization:** when the web visualizer is built, a JSON
  export is added as a bolt-on for the browser. It is not an engine-core
  commitment and does not constrain the API now.

---

## 10. Deferred layers

- **Language server (`lsp/`)** — deferred. The engine is *built for* it (the
  incremental query engine is the enabling machinery), but it is not in the first
  cut. It will be a later package consuming the same engine, not a rewrite.
- **Interchange / serialization** — deferred beyond the single browser-facing viz
  export noted in §9. Full XMI/YAML/JSON-LD/KPAR round-trip (syster's
  `interchange`) is out of scope for e0.1.

---

## 11. Attribution note

syster is MIT-licensed. This engine is a from-scratch Go reimplementation
informed by syster's architecture, not a code port. License and attribution
details should be confirmed against the upstream terms before release; this note
is not legal advice.

---

## 12. Decision log

- **E1.** Pure-Go reimplementation with full semantic resolution; mirror syster's
  architecture.
- **E2.** Rejected cgo, Rust→Wasm/wazero, subprocess CLI, REST API server,
  Syside shell-out, and parser-only ANTLR/tree-sitter (§3).
- **E3.** Layer 1: full rowan/cstree-style red/green lossless CST.
- **E4.** Layer 2: from-scratch Salsa port (not go-incr, not gopls-style
  hand-rolled).
- **E5.** Layer 3: full KerML/SysML semantic fidelity (deep inheritance + stdlib),
  not a structural subset.
- **E6.** Layer 4: SysML v2 standard library embedded via `go:embed` and indexed.
- **E7.** Public surface: in-process typed Go API over the resolved model; no
  serialized model in core.
- **E8.** LSP deferred to a later package on the same engine.
- **E9.** Interchange deferred except a future browser-facing viz JSON export
  (bolt-on).
- **E10.** Concurrency: fine-grained concurrent execution; goroutine-safe memo
  store; immutable green nodes shared without refcounting (GC-managed).
- **E11.** Green-node storage: arena/index-based; red tree computed on demand as
  cursors (not persisted).
- **E12.** Parser: hand-written recursive-descent mirroring syster (not ANTLR,
  tree-sitter, or combinators).

---

## 13. Open / verification items

- Concurrent green-node construction strategy: synchronized shared arena + intern
  map **vs** per-goroutine arenas merged at the end — to be chosen and benchmarked
  during Layer 1/5a work.
- GC-pressure validation of the arena/index store at model scale (millions of
  nodes) — to be measured.
- Cycle handling and cancellation semantics in the query engine — to be designed
  during Layer 2 work.
- Mapping table (§2) LOC figures are approximate, from static inspection of
  syster; treat as sizing guidance, not exact targets.

# Engine concurrency model

This note records the concurrency decisions for the engine and the resolution of
the open/verification items in `specs/ENGINE.md` §13 (concurrency model §5a,
decision E10).

## Chosen strategy: independent per-file construction

ENGINE §13 left open: *"synchronized shared arena + intern map **vs**
per-goroutine arenas merged at the end — to be chosen and benchmarked."*

**Decision: fully independent per-file construction — neither a shared arena nor
a merge step.**

- Each file is parsed by its own `parser.Parse`, which builds its own
  `cst.Builder` with its own green arena **and** its own interner. During parsing
  there is no shared mutable state at all, so no synchronization is needed and
  construction scales linearly with cores.
- `hir.AnalyzeUnits` parses all units concurrently (`parseUnits`, bounded to
  `GOMAXPROCS`) into a `[]ast.SourceFile`, writing each result to a distinct
  slice index (race-free), then builds the combined symbol model sequentially.
  The combined model is therefore **deterministic** regardless of scheduling.
- Cross-file identity is established later, by name, in the symbol/resolution
  layer — not by sharing interned ids across trees. This keeps the hot
  construction path lock-free.

Rejected alternative — a single synchronized arena + intern map shared across
goroutines: it adds lock contention on the hottest path (node/token append) for
little benefit in the batch-generation use case, where files are independent and
cross-file green-node sharing is rare. A shared, goroutine-safe `text.Interner`
is still available for callers who want cross-tree id sharing (it uses an
`RWMutex`); `TestConcurrentSharedInterner` exercises that path.

Immutable green nodes are shared across goroutines without refcounting — the GC
manages their lifetime (E10), so the read-side (red cursors, `Node.Text`, typed
AST, resolution) needs no locks.

## Incremental query engine (salsa)

The memo store is **goroutine-safe by serialization**: `salsa.Db` guards all
state under one mutex, and each `Db.Read` holds it for the duration of a
dependency-tracked session. This is the first-cut strategy sanctioned by
ENGINE §13; finer-grained locking is a future optimization and does not change
the public API (the immutable-result core makes it a mechanical change).

- **Cancellation** — reads honor `context.Context`; a cancelled context unwinds
  to `Db.Read` as the context's error. Verified concurrently by
  `TestConcurrentCancellation`.
- **Cycles** — a query that transitively reads itself is detected and surfaces
  as a `CycleError` (`TestCycleDetection`).
- **Concurrent readers + writers** — `TestConcurrentReadsAndWrites` runs the
  engine under `-race` with interleaved `Set`/`Read`.

## GC-pressure validation

The green store is arena/index-based (typed slices; children by integer index,
not pointers), chosen to keep GC pressure low at scale. Measure it with:

```
go test -run '^$' -bench BenchmarkBuildTree -benchmem ./engine/cst/
```

`BenchmarkBuildTree` builds a 1000-node tree; `-benchmem` reports bytes and
allocations per build. Absolute numbers are environment-dependent; the arena
design keeps allocations proportional to node count with no per-node pointer
overhead. Related benchmarks: `BenchmarkTraverseText` (cst),
`BenchmarkQueryHit` / `BenchmarkInvalidation` (salsa), `BenchmarkAnalyzeUnits`
(hir).

## Race verification

The whole engine is exercised under the race detector in CI
(`go test -race ./...`), including the concurrency tests above.

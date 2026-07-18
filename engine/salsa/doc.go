// Package salsa is Layer 2 of the sysgo engine (specs/ENGINE.md §6): a
// from-scratch incremental query engine, a Go port of Salsa's algorithm.
//
// The model has two kinds of query:
//
//   - Inputs ([Input]) are durable base facts set from outside. Setting an input
//     to a new value bumps a global [Revision] and stamps the input as changed
//     at that revision. Setting it to an equal value is a no-op (no bump).
//   - Tracked queries ([Query]) are pure, memoized functions of the database.
//     While a query computes, every input or query it reads is recorded as a
//     dependency. The result is cached with a verified-at and a changed-at
//     revision.
//
// On each read the engine verifies the cached value bottom-up: if no dependency
// has changed since the value was last verified, the cache is reused (only its
// verified-at is advanced). Otherwise the query is recomputed — and if the new
// result equals the old one, its changed-at is left untouched (backdating), so
// dependents that only cared about the value are not recomputed. This is what
// makes recomputation minimal: an input edit only re-runs the queries whose
// results actually change.
//
// Cycles are detected (a query that transitively reads itself) and surface as a
// [CycleError]. Reads honor a context.Context; a cancelled context surfaces the
// context's error. A [Db] is safe for concurrent use — reads and writes are
// serialized under one lock (fine-grained concurrency is ENGINE §13 / a later
// hardening pass); the immutable-result core keeps that a mechanical change.
package salsa

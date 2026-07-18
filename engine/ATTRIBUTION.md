# Attribution

The sysgo engine is a **from-scratch Go reimplementation** whose architecture is
informed by **syster**, a Rust SysML v2 / KerML toolchain:

- `syster` — `github.com/jade-codes/syster-base`, `github.com/jade-codes/syster-cli`
- License: **MIT**

This engine reproduces syster's *design* (a rowan/cstree-style lossless red/green
CST, a hand-written recursive-descent parser, a Salsa-style incremental query
engine, and a HIR semantic layer) rather than porting its code; the two are
written in different languages. See `specs/ENGINE.md` §2 (layer map) and §11 for
the rationale.

Salsa (the incremental-computation framework whose algorithm `engine/salsa`
ports) is developed at `github.com/salsa-rs/salsa` (Apache-2.0 / MIT).

The bundled standard-library files under `engine/project/stdlib/` are a curated,
parser-compatible subset derived from the **OMG SysML v2 release** (the SysML-v2
Pilot Implementation and standard library are MIT-licensed). Each file carries a
header noting its provenance.

> This note records provenance and is **not legal advice**. License and
> attribution details should be confirmed against the upstream terms before any
> distribution or release (ENGINE §11, §13).

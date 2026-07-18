# TEMPORAL — sysgo Temporal Spec

**Version:** s0.2
**Status:** Decoupled from consumers; all s0.1 open decisions resolved.
**Parent:** `OVERVIEW.md`. **Depends on:** `ENGINE.md`,
`TEST.md` (replay tests).

---

## 1. Purpose and scope

Generates Temporal Go (workflows, activities, workers, signals, schedules) from
**native SysML v2 actions** annotated by a **Temporal metadata profile**. The
model is the single source of truth (framework §2); this generator is a `gen/`
consumer of the engine's resolved model via the in-process typed Go API
(framework §4).

**Decoupling (new in s0.2):** this spec defines only the profile, the
action→Temporal mapping, and generation. **Consumer-specific action mappings do
not live here** — Codiq's (and any Epos) workflow/activity annotations live in the
respective consumer specs. s0.1's embedded Codiq mapping has been removed.

---

## 2. Representation — native actions + metadata profile

Temporal is modeled with native SysML v2 actions plus a `TemporalProfile`
metadata profile (`metadata def`s — standard-compliant, no grammar fork). The
profile ships **bundled and embedded with the engine** like the SysML standard
library (`go:embed`), not vendored per project (resolves s0.1 #5).

```
package TemporalProfile {
    metadata def Workflow        { attribute id : String; attribute taskQueue : String; }
    metadata def Activity        { attribute taskQueue : String; }
    metadata def Signal          { attribute name : String; }
    metadata def Query           { attribute name : String; }
    metadata def RetryPolicy     { attribute maxAttempts : Integer; attribute initialInterval : Duration;
                                   attribute backoffCoefficient : Real; attribute maxInterval : Duration; }
    metadata def Timeout         { attribute startToClose : Duration; attribute scheduleToClose : Duration;
                                   attribute heartbeat : Duration; }
    metadata def Deterministic;                    // orchestration-only, no I/O
    metadata def Idempotent;                        // activity retry-safety contract
    metadata def Schedule        { attribute spec : String; attribute jitter : Duration; }
    metadata def ExternalStorage { attribute threshold : Integer; }   // claim-check + encrypting codec
}
```

---

## 3. Action → Temporal mapping

- Composite `action def` + `@Workflow` → workflow function.
- Atomic `action def` + `@Activity` (+ `@RetryPolicy`/`@Timeout`/`@Idempotent`,
  optional `taskQueue`) → activity function + registered `ActivityOptions`. The
  activity **interface is the hexagonal outbound port**; the adapter implements
  the side effect.
- `accept signal: X` / `send` → `@Signal` channel handlers
  (`workflow.GetSignalChannel`).
- `state def` + transitions → state-machine workflow (selector loop).
- `@Schedule` on a workflow → a Temporal Schedule.
- `@ExternalStorage` → encrypting Payload Codec + external (S3) data converter
  wiring for that payload.
- `perform` (part → action) → task-queue-scoped worker registration.

### 3.1 Control-flow fidelity (resolves s0.1 #7 — full mapping)

The v1 workflow-body generator maps the **full** behavioral surface:

- succession → sequential activity calls;
- `fork`/`join` → parallel activity futures;
- guarded successions / decision·merge → `if/else`;
- repetition → `for`;
- `accept after` / `accept at` → durable timers (`workflow.Sleep` / timer
  futures);
- `accept signal` → signal handlers; `state def` → selector-loop state machines.

**Risk (tracked):** SysML v2 control-node semantics are not fully settled, and a
succession asserts *precedence*, not *triggering*. Mapping guarded
successions / merge / timers to durable Temporal control flow has real edge
cases. The engine's full semantic resolution (engine §7) supplies a precisely
resolved behavioral model, and jennifer (§5) makes the control-flow emission
tractable, but **control-node → durable-Temporal mappings must be validated
against edge cases** before relying on them.

---

## 4. Generated artifacts and generation boundary

Generated (regenerated wholesale; framework §2):

- workflow functions (fully generated from action structure — resolves s0.1 #1);
- activity **interfaces** (= outbound ports) + `ActivityOptions` + registration;
- worker `main` + task-queue-scoped registration (from `perform`);
- signal/query channel handlers;
- schedule creation code;
- data converter + encrypting Payload Codec wiring (for `@ExternalStorage`).

Hand-written (never clobbered):

- activity **bodies** — emitted **once as stubs** if absent, then owned by the
  author, in separate files from generated code. This is the identical
  generated-interface / hand-written-adapter split used by the K8s CRUD outbound
  port and gqlgen resolver bodies (resolves s0.1 #2).

Boundary: generated workflow bodies invoke only activity ports and Temporal APIs;
all side effects live in hand-written activity adapters.

---

## 5. Emitter (framework convention)

Go is emitted with **jennifer** (`github.com/dave/jennifer`); any YAML is emitted
with `text/template` (framework §5a). jennifer's programmatic construction and
automatic import management suit Temporal's control-flow-heavy, import-heavy
workflow bodies. (Resolves s0.1 #4; the mechanism is framework-wide, not
Temporal-specific.)

Model parsing/resolution is provided by the engine (resolves s0.1 #3); there is
no Temporal-specific parser.

---

## 6. Determinism enforcement (resolves s0.1 #6 — A+B+C)

- **A (baseline, by construction):** fully-generated workflow bodies emit only
  deterministic constructs (activity calls, workflow APIs, durable timers); side
  effects are confined to hand-written activities.
- **B (build-time):** integrate Temporal's official `workflowcheck`
  (`go.temporal.io/sdk/contrib/tools/workflowcheck`, a `go/analysis` analyzer) as
  a `vet`-style check in the **generated CI** (framework §7). Guards hand-written
  and hand-edited workflow code. Note: `workflowcheck` is a helper and does not
  catch every case (e.g. global-var mutation).
- **C (run-time):** the test generator (`TEST.md`) emits Temporal
  **replay tests** that detect history drift. Referenced, not duplicated here.

---

## 7. Decision log

- **T1.** Representation: native actions + `TemporalProfile` metadata profile
  (no grammar fork).
- **T2.** Profile bundled/embedded with the engine (`go:embed`), not vendored.
- **T3.** Full action→Temporal control-flow mapping in v1 (linear, fork/join,
  branch, loop, durable timers, signals, state machines).
- **T4.** Workflow bodies fully generated; activity interfaces generated;
  activity bodies hand-written as stubs, never clobbered.
- **T5.** Emitter: jennifer for Go, `text/template` for YAML (framework §5a).
- **T6.** Model parsing via the engine; no separate parser.
- **T7.** Determinism enforced A+B+C: by-construction baseline, `workflowcheck`
  in CI, replay tests from the test generator.
- **T8.** Consumer-specific action mappings live in consumer specs, not here.

---

## 8. Open / risk items

- Control-node → durable-Temporal mapping edge cases (§3.1) — validate before
  relying on branch/merge/timer generation.
- `workflowcheck` configuration (ident-refs overrides, false positives) — tune
  per project.
- Versioning/patching strategy for evolving workflows (Temporal `GetVersion` /
  patched) vs wholesale regeneration — to be designed (interaction between
  model-driven regeneration and Temporal's determinism-preserving versioning).

# SYSGO-FRAMEWORK-SPEC

**Version:** f0.1
**Status:** Foundational decisions locked; per-area generator specs pending.
**Role:** Thin umbrella. Holds only cross-cutting principles, platform structure, and shared conventions. Deep designs live in referenced specs.

---

## 1. Scope and spec organization

This is the umbrella spec for a model-centric, shift-left framework in which a
single SysML v2 model is the source of truth for generated code, contracts,
infrastructure, reliability objectives, and tests.

The framework is intentionally split across files. This document is deliberately
thin. It is referenced by, and does not duplicate, the following:

- `SYSGO-ENGINE-SPEC.md` — the pure-Go SysML v2 engine (parser, incremental
  query engine, semantic resolution, standard library, public API).
- `SYSGO-VIZ-SPEC.md` — visualization architecture (client-side Go→Wasm, Guix
  render surface, pure-Go layout). Depends on Guix
  (`github.com/gaarutyunov/guix`); its detailed build-out is a forthcoming
  Guix-update spec.
- `SYSGO-TEMPORAL-SPEC.md` — Temporal Go generation from SysML behavioral
  actions plus a Temporal metadata profile (refreshed to s0.2: decoupled from
  consumers, all open decisions resolved).
- Per-area generator specs (pending, one decision round each):
  `SYSGO-CONTRACTS-SPEC.md`, `SYSGO-K8S-SPEC.md`, `SYSGO-DEPLOY-SPEC.md`,
  `SYSGO-SLO-SPEC.md`, `SYSGO-TEST-SPEC.md`.

Naming/versioning follows the existing sysgo convention: one file per
capability, single-letter version prefix (`f` framework, `e` engine, `s`
temporal, and per-area prefixes to be assigned).

---

## 2. Foundational principle — the model is always the source of truth

Every generated artifact is an **output** of the model. No generated artifact is
canonical; if it needs to change, the model changes and the artifact is
re-emitted. This is a hard, framework-wide constraint and it governs every
per-area spec.

Consequence for testing that supersedes prior project discipline: `.feature`
files are generated outputs, re-emitted wholesale from use-case actions — they
are **not** hand-authored or canonical. **Epos is unified onto this principle**:
its earlier rule that feature files are the canonical, never-generated test
source is replaced by model-generated features. The Epos testing section must be
updated to match; until then, the two specs are known to disagree on this point
and this spec takes precedence.

---

## 3. Platform structure

The framework is a three-layer platform in **one repository, one Go module**:

- `engine/` — the reusable SysML v2 engine (see `SYSGO-ENGINE-SPEC.md`).
- `gen/` — the generators (the per-area capabilities).
- `viz/` — LikeC4-capable visualization (later; see §7).
- `internal/` — used to enforce boundaries between the layers.

Single `go.mod` at the repo root; one version; one CI. Cross-layer refactors are
atomic. The engine remains a reusable library *by import path and package
boundary*, not by independent versioning — a deliberate trade of independent
release for single-version simplicity.

**Downstream consumers:** codiq and Epos both depend on this module.

---

## 4. Layer contract

Downstream layers consume the engine through an **in-process, typed Go API over
the resolved model**. There is no serialized model in the common path.
Generators, codiq, and Epos import the engine and call it directly.

Serialization is deliberately kept out of the engine's core surface. When the
web visualizer is built it grows its own JSON export as a bolt-on for the
browser; the engine does not commit to a stable serialized model contract now.

---

## 5. Profile and metadata conventions

External artifact types are represented in the model using **SysML v2 metadata
definitions** (the v2 replacement for v1 stereotypes), applied to native
constructs (`item def`, `part def`, `port def`/`interface def`, `action def`,
`state def`, enumeration defs, `package`).

Two conventions apply across all per-area specs:

1. **Representation-in-model is required.** Each artifact type must be
   expressible in the model, not merely emitted as a side output.
2. **Per-protocol separation for contracts.** The OpenAPI, GraphQL, and protobuf
   profiles are distinct and each models its protocol's full concepts natively
   (full fidelity, accepted duplication when one API is exposed over two
   protocols). Detail lives in `SYSGO-CONTRACTS-SPEC.md`.

The "how much does the model own vs. what does the emitter default" boundary is
set **per area** (it differs between, e.g., deployment and CI) and is recorded in
each per-area spec, not here.

### 5a. Emitter convention (framework-wide)

All generators use one emitter convention:

- **Generated Go** is emitted with **jennifer** (`github.com/dave/jennifer`,
  MIT) — programmatic AST construction with automatic import management and
  `gofmt`-correct output, well-suited to control-flow-heavy generated code.
- **Generated YAML** (CRDs, Knative, Skaffold, OpenSLO manifests, etc.) is
  emitted with Go `text/template`.

This is a hybrid by output type (Go → jennifer, YAML → templates) and applies to
every per-area generator.

---

## 6. Deployment convention (standing rule)

Applies uniformly to every generated project and to this repo's own preview/docs
sites:

- **GitHub Pages: branch mode** — `gh-pages` branch via
  `peaceiris/actions-gh-pages`.
- **PR preview deployments** via `rossjrw/pr-preview-action` with `action: auto`.
- For projects with a web frontend (e.g. `viz`), **Vite `base` must be `'./'`**
  (relative paths) for PR-preview subpath compatibility.
- **Data artifacts must never use Git LFS** (GitHub Pages does not serve LFS
  objects).

This rule is emitted into whatever CI scaffolding is generated (see §7).

---

## 7. CI convention

CI is **fixed scaffolding, not modeled.** sysgo emits a well-formed workflow
once — the §6 deployment rule baked in, wired to the generated build/test/deploy
targets — and it is hand-maintained thereafter. CI pipeline structure is not
authored in the model.

This is consistent with, not contrary to, "model owns nearly everything" for
deployment. The boundary is deliberate and must be stated as such in the deploy
spec:

- **Deployment/serve intent is modeled** — the Knative/Skaffold surface (what is
  deployed, how it scales, where it serves).
- **The CI harness that executes build → test → deploy is fixed infrastructure**
  — runner tuning, caching, secrets, action versions live in YAML, where they
  naturally change.

---

## 8. Decision log

- **F1.** Model is always the single source of truth; all generated artifacts are
  outputs. Epos unifies onto this (features generated, not canonical).
- **F2.** One repo, one Go module; packages `engine/`, `gen/`, `viz/`;
  `internal/` guards boundaries.
- **F3.** Engine is a reusable library by package boundary, not independent
  version (single-module monorepo).
- **F4.** Downstream layers consume the engine via an in-process typed Go API
  over the resolved model; no serialized model in the common path.
- **F5.** Serialization stays out of the engine core; web viz adds a JSON export
  bolt-on later.
- **F6.** Artifact types are represented via SysML v2 metadata definitions on
  native constructs; representation-in-model is required.
- **F7.** Contract profiles are per-protocol and separate (full fidelity).
- **F8.** Standing deployment rule (§6) applies to all generated projects.
- **F9.** CI is fixed scaffolding, not modeled; deployment intent is modeled, the
  CI harness is fixed.
- **F10.** Spec organization: thin umbrella + dedicated engine spec + per-area
  specs.
- **F11.** Emitter convention (framework-wide): jennifer for generated Go,
  `text/template` for generated YAML (§5a).

---

## 9. Open / deferred

- Per-area generator specs (contracts, k8s, deploy, slo, test) — each pending its
  own decision round.
- Visualization layer (`viz/`) — architecture now locked (see
  `SYSGO-VIZ-SPEC.md`); the concrete Guix build-out (layout engine, diagram
  components, interaction, export) is a forthcoming Guix-update spec.
- Language server — deferred (see `SYSGO-ENGINE-SPEC.md`).

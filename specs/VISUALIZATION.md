# VISUALIZATION — sysgo Visualization Spec

**Version:** v0.1
**Status:** Architecture locked. Concrete Guix build-out deferred to a separate
Guix-update spec.
**Parent:** `OVERVIEW.md`. **Depends on:** `ENGINE.md`.

---

## 1. Purpose and bar

`viz/` provides interactive diagram visualization of SysML v2 models. The
explicit capability bar is **"as capable as LikeC4"**: auto-generated,
interactive, multi-level diagrams with dynamic views, styling, and navigation,
staying always in sync with the model.

This spec fixes the **architecture**. The concrete implementation work inside
Guix (see §3, §4) is specified separately in a forthcoming **Guix-update spec**.

---

## 2. Pipeline location — client-side Go→Wasm

The engine and `viz/` compile to **WebAssembly and run entirely client-side.**
Resolution (via the engine) and layout (§4) both run in the browser; there is no
server-side layout step and no server dependency for rendering.

- *Advantage:* one Go codebase runs in the browser; live and local; own-the-stack.
- *Accepted cost:* layout must run in Wasm (addressed by §4); Go→Wasm bundle size
  and Canvas/DOM interaction ergonomics are managed via Guix (§3).

Because engine and viz share one Wasm binary, viz consumes the engine's
**in-process typed Go API** over the resolved model directly (per framework §4
and engine §9). The "bolt-on JSON export" noted in engine §9 is only relevant if
viz is ever decoupled from the engine binary; in the client-side model they are
the same binary, so direct API use is the norm.

---

## 3. Render / interaction surface — Guix

Rendering and interaction use **Guix** (`github.com/gaarutyunov/guix`, MIT) — a
pure-Go UI framework that transpiles a `.gx` DSL to WebAssembly.

Guix already provides the two hard rendering primitives this needs:

- a **virtual-DOM / reactive UI** layer (channel-based state, keyed
  reconciliation, event handling) for panels, controls, navigation, and
  dynamic-view stepping; and
- a **WebGPU** path with GPU-accelerated 2D charts and scene graphs for a
  high-performance diagram canvas.

Diagrams are rendered through Guix; no JavaScript rendering framework
(xyflow/react-flow, D3) is used.

---

## 4. Layout engine — pure-Go, built in Guix

Graph layout (node positioning + edge routing) is implemented **in pure Go,
inside Guix**, compiled into the same Wasm binary. No `elkjs`, Graphviz, or other
JavaScript/native layout dependency.

Target capability (ELK-layered / Sugiyama class), because SysML models require
it:

- **layered (Sugiyama) hierarchical layout** for directed structure;
- **port-awareness** — SysML interfaces/parts expose ports as explicit attachment
  points;
- **compound / nested graphs** — `part def`s contain parts; packages nest;
- **edge routing** (orthogonal where appropriate).

- *Advantage:* fully pure-Go/Wasm, one binary, deterministic, offline, no JS
  dependency in the bundle — consistent with the platform's own-the-stack line.
- *Accepted cost:* this is the single largest deferred workstream in the
  platform. ELK-grade layered layout is algorithmically deep (ELK carries ~140
  options); existing pure-Go layout libraries (e.g. `gverger/go-graph-layout`)
  are early and do not yet cover port-aware, nested, orthogonally-routed layout.

**Rejected:** `elkjs` via JS interop (ELK-grade for free, but a JavaScript
dependency in the client bundle, against the C decision); Graphviz via
`goccy/go-graphviz` (Wasm-in-Wasm in the browser, dubious viability) or
Graphviz-wasm via interop (weaker port/nesting handling than ELK, plus the JS
dependency).

---

## 5. View selection — native SysML v2 views

Diagram content is selected using SysML v2's **native** `view def` /
`viewpoint def` / `expose` / `rendering` constructs (framework §5 advantage),
rather than a bespoke view DSL. LikeC4-style include/exclude/of predicates map
onto `expose` and viewpoint framing.

---

## 6. Deployment

Follows framework §6 (GitHub Pages branch mode via `peaceiris/actions-gh-pages`;
PR preview via `rossjrw/pr-preview-action`, `action: auto`).

Note: the viz build is Guix (`guix generate` + `GOOS=js GOARCH=wasm go build`),
not Vite — so the framework's "Vite `base: './'`" clause does not apply
literally. The intent behind it still does: the Wasm/JS glue and asset references
must use **relative paths** so PR-preview subpath hosting works.

---

## 7. Decision log

- **V1.** Pipeline runs client-side; engine + viz compile to Go→Wasm.
- **V2.** Render/interaction surface is Guix (VDOM + WebGPU); no JS render
  framework.
- **V3.** Layout is pure-Go, built into Guix (ELK-layered/Sugiyama-class target);
  no elkjs/Graphviz/JS dependency.
- **V4.** View content is selected via native SysML v2 view/viewpoint/expose
  constructs.
- **V5.** viz uses the engine's in-process Go API directly (same Wasm binary); no
  serialized model needed unless later decoupled.

---

## 8. Deferred — the Guix-update spec

A separate spec will enumerate everything Guix must implement for a full diagram
visualization framework, including at least:

- the layered layout engine (§4) — port-aware, compound/nested, edge routing;
- diagram components — nodes, edges, ports, nested containers, labels, styling;
- viewport interaction — pan, zoom, expand/collapse, selection, focus;
- dynamic-view stepping (interaction flows);
- viewpoint → diagram rendering from the engine's resolved model;
- static export (SVG / PNG);
- large-model performance and virtualization.

Open items: export formats and fidelity; performance targets on large models;
whether any diagram types reuse Guix's existing GPU chart path.

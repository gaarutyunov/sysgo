# SPEC.md — `sysgo`: A SysML v2 → Go DDD/Hexagonal Code Generator

> Status: **Draft v0.2** · Working name: **`sysgo`** (placeholder — rename freely) · Module: `github.com/gaarutyunov/sysgo`
> 
> `sysgo` ingests a **SysML v2 model**, normalizes it into a **DDD/hexagonal intermediate representation**, and emits a **configurable Go project scaffold** whose layout follows the canonical Evans / Cockburn / Martin architecture. Its UX (config file, templates, overlays) is modeled directly on **oapi-codegen**.

-----

## 1. Motivation

We author systems models in SysML v2 (structure, ports, interfaces, actions, requirements). We want those models to be the *source of truth* that drives a Go codebase, the same way an OpenAPI spec drives `oapi-codegen` output — but instead of HTTP client/server stubs, we generate a **Domain-Driven, hexagonal Go scaffold**: domain entities/aggregates at the center, application use cases around them, ports as interfaces, adapters at the edges.

Three properties are non-negotiable:

1. **Spec-driven, regenerable.** Re-running the generator on an updated model produces an updated scaffold idempotently, without clobbering hand-written code.
1. **oapi-codegen-style UX.** A YAML config file, user-overridable `text/template` templates, and an **OpenAPI-Overlay-style** mechanism to patch the model before generation without editing the source.
1. **Proper layout.** The generated project obeys the Dependency Rule (dependencies point inward), with bounded-context-first (“Screaming”) top-level structure.

### Goals

- Consume a SysML v2 model via the **standard REST/HTTP API JSON** (primary) or an exported JSON file (offline).
- Produce a Go project laid out by **bounded context → four architectural regions** (Domain, Application, Adapters, Infrastructure).
- Make the **output directory structure and package names fully configurable** via rules, with **per-element overlays** for exact placement/overrides.
- Support **user template overrides** and **idempotent regeneration** with a generated/hand-written split.

### Non-Goals (v1)

- Parsing the SysML v2 **textual notation** *inside `sysgo`* (no production-grade Go parser exists). `sysgo` consumes the API JSON. To start from textual `.sysml` source, transform it to that JSON first with **real SysML tooling** — the repo ships `scripts/sysml2json.sh`, which drives the OMG **SysML v2 Pilot Implementation** serializer (`org.omg.sysml.xtext.util.SysML2JSON`); examples and tests are authored in `.sysml` and converted this way (see §5.1, §17, D-06).
- Generating runtime behavior/business logic. We generate **structure, signatures, ports, wiring, and stubs**; business rules are filled in by humans.
- Round-tripping Go → SysML (one-way generation only).
- Non-Go targets (the IR and a protoc-style plugin boundary keep this open for later — see §13).

-----

## 2. Background & Prior Art

|Tool            |Lesson we borrow                                                                                                                                                                                                                                               |
|----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
|**oapi-codegen**|Pipeline (parse → IR → `text/template`); YAML config with `generate` flags, `output-options`, `import-mapping`, `additional-imports`; user-overridable templates; **OpenAPI Overlay** support; `x-go-type`/`x-go-name` extension fields. **Primary reference.**|
|**goa**         |Design-first DSL → clean layered `gen/` tree; the **regenerate-vs-scaffold-once** split (`goa gen` always overwrites; `goa example` scaffolds once); plugin hooks that mutate the design and file set.                                                         |
|**ent**         |External `--template` override (same-name overrides built-in, else new file); `Annotations` (JSON-serializable metadata on schema objects) ≈ our element metadata; standard `// Code generated … DO NOT EDIT.` header.                                         |
|**protoc / buf**|The decoupled **plugin contract**: read a `CodeGeneratorRequest`, write a `CodeGeneratorResponse{File{name,content}}`. Our future backend-plugin boundary.                                                                                                     |
|**sqlc**        |WASM/`process` codegen plugins; `version: 2` config with per-output `overrides` and `go_type` mapping.                                                                                                                                                         |

**SysML v2** has graphical + textual notation and a standardized REST/HTTP API. The reference pilot implementation is Xtext/Java/ANTLR-based; there is **no mature Go parser**. The pragmatic ingestion path is the **API JSON** serialization (`@type`, `@id`, `ownedRelationship` → `Membership` → `ownedRelatedElement`).

**The canonical layout** (Evans *DDD*, Cockburn *Ports & Adapters*, Martin *Clean Architecture*, Palermo *Onion*, Vernon *IDDD*) reduces to one rule — **domain at the center, infrastructure at the edges, all source-code dependencies point inward** — across four regions. §9 specifies how we emit it.

-----

## 3. Ubiquitous Language (terminology used in this spec)

- **Model** — a SysML v2 project at a specific commit; a graph of **Elements**.
- **Element** — a SysML v2 node (`PartDefinition`, `PortDefinition`, `ActionDefinition`, `Package`, …) identified by `@id`/`elementId`, with `@type` and containment via memberships.
- **IR (Intermediate Representation)** — `sysgo`’s own DDD model (`Context`, `Entity`, `ValueObject`, `Port`, `UseCase`, …), decoupled from the SysML metamodel.
- **Region** — one of the four architectural layers we emit into: **Domain**, **Application**, **Adapter**, **Infrastructure**.
- **Driving (primary) port/adapter** — the inbound boundary (use-case interfaces; HTTP/CLI handlers that call them).
- **Driven (secondary) port/adapter** — the outbound boundary (repository/gateway interfaces; DB/HTTP implementations).
- **Overlay** — an OpenAPI-Overlay-style document of `actions` (`target` JSONPath + `update`/`remove`/`copy`) applied to the model JSON **before** IR construction.
- **Bounded Context** — a SysML `Package` (by default) mapped to a top-level Go module subtree.

-----

## 4. High-Level Architecture

```
            ┌──────────── sysgo ────────────────────────────────────────────┐
SysML v2 ──►│ Loader ──► Overlay Engine ──► IR Builder ──► Renderer ──► Emitter │──► Go project
 (API/JSON) │  (graph)     (JSONPath)        (DDD model)   (templates)  (gofmt)  │   (scaffold)
            └────────────────────────────────────────────────────────────────┘
                                            ▲
                                   config.yaml + overlay.yaml + user templates
```

Stages:

1. **Loader** — fetch/read the SysML v2 element graph; resolve memberships into a navigable tree keyed by `@id`.
1. **Overlay Engine** — apply user overlay actions to the raw model JSON (inject metadata, relocate/remove elements) before anything is interpreted.
1. **IR Builder** — apply mapping rules (heuristics + metadata) to produce the DDD IR; this is where Go-specific decisions (type, package, pointer-ness, stereotype) are resolved.
1. **Renderer** — execute embedded (or user-overridden) `text/template` templates against IR nodes.
1. **Emitter** — write files into the configured layout, run `gofmt`/`goimports`, apply the generated/scaffold-once policy, and (optionally) verify freshness.

Each stage is an interface (see §13) so the loader, overlay engine, renderer, and writer are independently swappable and testable.

-----

## 5. Input: SysML v2 Ingestion

### 5.1 Sources

- **API mode (primary).** Point `sysgo` at a SysML v2 API Services base URL + `projectId` + `commitId`. It pages through `GET /projects/{p}/commits/{c}/elements` (and fetches individual elements / runs `POST /query-results` as needed).
- **File mode (offline).** A pre-exported JSON array of elements (the API’s serialization), e.g. produced by a prior export or a CI artifact.
- **Textual source via tooling (authoring).** Models are authored in the SysML v2 **textual notation** (`.sysml`) and transformed to the API JSON *before* `sysgo` runs, by real SysML tooling — `sysgo` ingests the JSON, never the text. The canonical path is the OMG **Pilot Implementation** serializer `org.omg.sysml.xtext.util.SysML2JSON` (bundled in the pilot’s Jupyter-kernel distribution), wrapped by `scripts/sysml2json.sh` (`make model`). This is the source of the example/test fixtures (§17). Output element `@id`s are not stable across tool runs, but `sysgo`’s generated output does not depend on them.

> Pin to a specific server/spec version. The published API document and generated clients show version skew (OAS 3.0.1 vs 3.1; `version: 1.0.0`); SysML v2 is still finalizing at OMG. Treat the live server’s document as ground truth.

### 5.2 The element graph

Each element is a JSON-LD-ish object:

```json
{
  "@id": "b7403610-e104-4b8e-9eac-7d4ee5c41de0",
  "@type": "PartDefinition",
  "declaredName": "Order",
  "elementId": "b7403610-e104-4b8e-9eac-7d4ee5c41de0",
  "ownedRelationship": [{ "@id": "8e57348e-338f-4272-9e01-bb896784d849" }]
}
```

Containment is **indirect**: an Element owns a **Membership** (`OwningMembership`, `FeatureMembership`, …) via `ownedRelationship`; the membership points to the contained Element via `ownedRelatedElement`. A derived `ownedElement` array, where present, lets us skip the membership hop. Cross-references are always `{"@id": "..."}`.

**Real serialization details the Loader/IR builder accommodate** (validated against the Pilot serializer output):

- **API envelope.** The pilot/REST bulk format wraps each element as `{ "payload": <element>, "identity": {"@id": …} }`. The Loader unwraps `payload`; plain element arrays pass through unchanged.
- **Typing by reference.** A feature’s type is not a name on the feature but a **`FeatureTyping`** relationship (in its `ownedRelationship`) whose `type` references the type element by `@id`. The IR builder dereferences it to the type’s (qualified) name.
- **Multiplicity by element.** Bounds are a **`MultiplicityRange`** owned by the feature, holding `LiteralInteger`/`LiteralInfinity` bound elements (a bare `Multiplicity` is the default `1..1`). `*` ⇒ many; lower `0` ⇒ optional.
- **Library proxies.** Cross-document references to standard-library types (e.g. `ScalarValues::String`) serialize as **anonymous `Type` proxies** unless the referenced library is included in the export. `scripts/sysml2json.sh` therefore bundles the needed library (e.g. `ScalarValues.kerml`) so scalar types emit as named `DataType` elements; library elements are used for type resolution but are never generated.

The Loader resolves this into an in-memory `model.Graph`:

```go
type Graph struct {
    Elements map[string]*Element   // keyed by @id
    Roots    []*Element            // RootNamespace / top-level packages
}
type Element struct {
    ID, Type, DeclaredName string
    Raw        map[string]any      // full JSON (overlays operate here)
    Owned      []*Element          // resolved children (via memberships)
    Owner      *Element
}
```

-----

## 6. Intermediate Representation (IR)

The IR is `sysgo`‘s domain model — the analogue of oapi-codegen’s `GoSchema` / goa’s expressions. Templates render **only** against the IR, never raw SysML.

```go
type Project struct {
    Module   string
    Contexts []*Context
}
type Context struct {            // ← SysML Package / bounded context
    Name, Package, Dir string
    Entities     []*Entity
    ValueObjects []*ValueObject
    DomainServices []*DomainService
    UseCases     []*UseCase
    DrivenPorts  []*Port          // repository/gateway interfaces
    DrivingPorts []*Port          // use-case boundary interfaces
    Events       []*DomainEvent
}
type Entity struct {
    Name string; Aggregate bool   // aggregate root?
    Fields []*Field
    Methods []*Method
    Meta Metadata                 // x-go-*, x-ddd-*
}
type Field struct { Name, GoType string; Optional, Pointer bool; Tags string }
type Port struct {
    Name string; Direction PortDir // In (driving) / Out (driven)
    Methods []*Method; Kind PortKind // Repository | Gateway | Service | UseCase
}
type UseCase struct { Name string; Input, Output *DTO; Port *Port }
```

`Metadata` carries resolved generation hints (`GoType`, `GoName`, `Tags`, `SkipOptionalPointer`, `Stereotype`, `TargetDir`, `TargetLayer`).

-----

## 7. SysML v2 → DDD Mapping

SysML has no DDD stereotypes natively, so mapping is **heuristics + explicit metadata** (metadata always wins).

### 7.1 Default heuristic mapping

|SysML element (`@type`)                                       |Heuristic                             |DDD IR target                         |Default region                       |
|--------------------------------------------------------------|--------------------------------------|--------------------------------------|-------------------------------------|
|`Package`                                                     |always                                |`Context` (bounded context)           |top-level                            |
|`PartDefinition` with an identity attribute / marked aggregate|identity present                      |**Aggregate root / Entity**           |Domain                               |
|`PartDefinition` value-like (no identity, immutable-ish)      |no identity                           |**Value Object**                      |Domain                               |
|`AttributeUsage`                                              |within a part def                     |**Field** (type-mapped)               |Domain                               |
|`PartUsage` (nested, via `FeatureMembership`)                 |composition                           |child Entity / composition field      |Domain                               |
|`PortDefinition`                                              |inbound vs outbound by item directions|**Port** (Go interface)               |Application (`port/in` or `port/out`)|
|`InterfaceDefinition` / `ConnectionDefinition`                |binds two ports                       |**Adapter contract**                  |Adapter                              |
|`InterfaceUsage` / `ConnectionUsage`                          |concrete binding                      |**Adapter** stub                      |Adapter                              |
|`ActionDefinition` / use-case element                         |application behavior                  |**UseCase** (interactor / app service)|Application                          |
|`RequirementDefinition`                                       |constraint                            |doc comment + test stub               |Domain/test                          |
|`ItemDefinition` / item flow                                  |data crossing a port                  |**DTO**                               |Domain/Application                   |

Port direction: an `in` item on a port → method **parameter**; an `out` item → method **return value**. A port whose items the application *receives requests on* is a **driving** port; one the application *calls outward through* is a **driven** port. Heuristic default: ports defined on an aggregate that reference persistence/external systems → driven; ports that expose actions → driving. Always overridable.

### 7.2 Explicit stereotyping (overrides heuristics)

Two mechanisms, both resolved before IR build:

1. **SysML metadata** — a `sysgo` metadata definition applied in-model (e.g. a `ddd` metadata def with a `stereotype` enum). Preferred when the modeler owns the model.
1. **Overlay-injected keys** — `x-ddd-stereotype`, `x-ddd-target-layer`, `x-ddd-target-dir`, `x-go-type`, `x-go-name`, `x-go-tags`, `x-go-skip-optional-pointer` injected via overlay `update`. Preferred when the consumer can’t edit the model (the oapi-codegen pattern).

-----

## 8. Type Mapping

OpenAPI-schema-style type mapping, keyed by SysML type qualified name:

- Built-in scalar map: SysML `ScalarValues`/ISQ types → Go (`Real`→`float64`, `Integer`→`int64`, `Boolean`→`bool`, `String`→`string`, etc.). Overridable in `type-mapping`.
- `x-go-type` (+ `x-go-type-import`) on an element forces an external Go type (e.g. `money.Money`).
- `x-go-skip-optional-pointer` suppresses the `*T` for optional fields (mirrors oapi-codegen’s `x-go-type-skip-optional-pointer`).
- `import-mapping` ties a SysML package / external reference to a Go import path; `additional-imports` injects extra imports per generated package.

-----

## 9. Generated Output Layout (the “proper” layout)

This is the heart of the spec. The emitted project obeys the **Dependency Rule** — every cross-region import points **inward** (Adapter → Application → Domain; nothing points outward). Structure is **bounded-context-first** (“Screaming Architecture”): the top-level directory names the business, not the framework.

### 9.1 The four regions and what lands in each

|Region                         |Depends on          |Contents (generated)                                                                                                                                    |
|-------------------------------|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
|**Domain** (center)            |nothing             |Entities, Value Objects, Aggregate roots, Domain Services, Factories, Domain Events. *(Optionally)* driven-port interfaces if configured `domain`.      |
|**Application** (around)       |Domain only         |Use Cases / Application Services (thin, no business rules); **driving port** interfaces (`port/in`); **driven port** interfaces (`port/out`) by default.|
|**Adapter**                    |Application + Domain|Driving adapters (HTTP/gRPC/CLI handlers) and driven adapters (repository/gateway **implementations**).                                                 |
|**Infrastructure / Frameworks**|edges               |Wiring, DB drivers, message brokers, the composition root (`cmd/`). “Details.”                                                                          |

**Repository placement** (the one genuine debate in the canon): the **interface is inward of the implementation**. Default: driven-port interfaces (incl. repositories) in `app/port/out`; implementations in `adapter/out/...`. A config knob (`ports.repository-in-domain: true`) moves repository interfaces into the Domain region for teams who prefer Evans’s framing.

### 9.2 Default emitted tree (per bounded context `order`, module `github.com/acme/orders`)

```
.
├── go.mod
├── cmd/
│   └── order/
│       └── main.go                      # composition root  (scaffold-once)
└── internal/
    └── order/                           # bounded context  (Screaming Architecture)
        ├── domain/                      # ── Region 1: Domain ───────────────
        │   ├── order.go                 # aggregate root            (gen)
        │   ├── order_line.go            # entity                    (gen)
        │   ├── money.go                 # value object              (gen)
        │   ├── order_events.go          # domain events             (gen)
        │   ├── order_factory.go         # factory                   (gen)
        │   └── pricing_service.go       # domain service: iface gen + impl scaffold
        ├── app/                         # ── Region 2: Application ──────────
        │   ├── port/
        │   │   ├── in/
        │   │   │   └── place_order.go    # driving port (use-case iface)  (gen)
        │   │   └── out/
        │   │       ├── order_repository.go   # driven port (repo iface)  (gen)
        │   │       └── payment_gateway.go     # driven port (gateway iface)(gen)
        │   └── usecase/
        │       └── place_order.go        # interactor: iface gen + impl scaffold
        └── adapter/                     # ── Region 3+4: Adapters / Infra ───
            ├── in/
            │   └── http/
            │       └── order_handler.go  # driving adapter           (scaffold)
            └── out/
                ├── postgres/
                │   └── order_repository.go   # driven adapter        (scaffold)
                └── stripe/
                    └── payment_gateway.go     # driven adapter       (scaffold)
```

Dependency directions (all inward): `adapter/out/postgres` imports `app/port/out` + `domain`; `app/usecase` imports `app/port/*` + `domain`; `domain` imports nothing in the context. `cmd/order` (composition root) is the only place allowed to import everything and wire concrete adapters into ports via constructor injection.

#### 9.2.1 Composition root — DI and cmd (two independent axes)

The composition root is not one-size-fits-all. Two orthogonal settings control
it — `generate.di` (*how* dependencies are wired) and `generate.cmd` (*which*
binaries exist). Every binary `main.go` is scaffold-once.

**`generate.di`** — optional [cobra](https://github.com/spf13/cobra) +
[wire](https://github.com/goforj/wire) wiring, independent of the cmd shape:

- **`enabled: false`** (default) — a minimal, dependency-free `main.go`
  (standard library only) that the user wires by hand. This keeps the default
  generated project buildable against the standard library alone.
- **`enabled: true`** — the binary main becomes a cobra entrypoint and, for
  every context, sysgo emits a **generated**
  `internal/<context>/providers.go` exposing a `wire.ProviderSet` that lists each
  constructor and `wire.Bind`s the concrete adapter/interactor to the port it
  satisfies. Each emitted binary additionally gets a **scaffold-once**
  `cmd/<name>/wire.go` injector (tagged `//go:build wireinject`); `wire ./...`
  in the generated project turns the stubs into concrete `wire_gen.go`. The
  arch-lint ruleset gains a `wiring` component so the context-root provider sets
  may compose every inner region, exactly like `cmd`. `provider` selects the
  toolkit; only `wire` is supported today. Because DI is independent of `cmd`,
  it may be enabled even with `cmd.mode: off` so a hand-written root can consume
  the provider sets.

**`generate.cmd.mode`** — which composition-root binaries are emitted:

- **`per-context`** (default) — one `cmd/<context>/main.go` per bounded context:
  a microservice per context, as shown in the tree above.
- **`mono`** — a single `cmd/<module>/main.go` wiring every context.
- **`custom`** — one binary per `cmd.groups` entry, each capturing a chosen set
  of contexts (`{ name, contexts: [...] }` → `cmd/<name>/main.go`).
- **`off`** — no `cmd/` files; the user composes the application by hand. Every
  other region is still generated.

### 9.3 Enforcing the Dependency Rule in the output

- Place region code under `internal/<context>/...` so Go’s `internal/` visibility plus import discipline keeps boundaries.
- Emit an optional **import-lint config** (e.g. a `depguard`/`go-arch-lint`/ArchUnit-for-Go ruleset) asserting no outward imports, so CI fails on violations.

-----

## 10. Configuration Schema

A single YAML file (`sysgo.yaml`), with a published JSON Schema referenced via an editor hint, mirroring oapi-codegen.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/gaarutyunov/sysgo/main/schema/sysgo.schema.json
module: github.com/acme/orders

source:
  api:                         # mode A
    base-url: http://localhost:9000
    project: 4f0e-...          # projectId
    commit:  9ab1-...          # commitId (omit ⇒ latest on default branch)
  # file: ./model.export.json  # mode B (mutually exclusive with api)

overlay:
  path: ./overlay.yaml         # applied to model JSON before IR build

generate:                      # which artifacts to emit
  domain:    true
  usecases:  true
  ports:     true
  adapters:  scaffold          # off | scaffold (stubs) | full
  events:    true
  tests:     false
  importlint: true             # emit arch-lint ruleset
  di:                          # dependency-injection wiring (independent of cmd)
    enabled: false             # emit wire ProviderSets + injectors
    provider: wire             # wire (only supported)
  cmd:                         # composition-root binaries
    mode: per-context          # per-context | mono | custom | off
    groups: []                 # custom mode: [{ name, contexts: [...] }, ...]

ports:
  driven-dir:  app/port/out
  driving-dir: app/port/in
  repository-in-domain: false  # move repo interfaces into domain region

layout:                        # region → directory + package; {context} interpolated
  domain:   { dir: "internal/{context}/domain",          package: domain }
  app:      { dir: "internal/{context}/app/usecase",     package: usecase }
  ports:    { dir: "internal/{context}/app/port",        package: port }
  adapters: { dir: "internal/{context}/adapter",         package: adapter }
  cmd:      { dir: "cmd/{context}",                       package: main }

type-mapping:
  ISQ::MassValue: { type: float64 }
  Currency::Money: { type: "money.Money", import: "github.com/acme/money" }

import-mapping:
  ScalarValues: github.com/acme/orders/pkg/scalars

additional-imports:
  - { package: github.com/google/uuid, alias: uuid }

output-options:
  user-templates:
    "domain/entity.go.tmpl": ./tmpl/entity.go.tmpl   # override one built-in
  skip-fmt:  false
  skip-prune: false
  generated-marker: "// Code generated by sysgo; DO NOT EDIT."
```

### Config precedence

`element metadata` > `overlay-injected keys` > `type-mapping`/`layout` rules > built-in defaults. (Most specific wins; this is what makes per-block overrides possible.)

-----

## 11. Overlay Mechanism

Reuse the **OpenAPI Overlay Specification** grammar verbatim, applied to the SysML model JSON (a flat array of elements) before IR build. Because elements carry `@id`/`@type`/`declaredName`, JSONPath selectors are natural.

```yaml
overlay: 1.0.0
info: { title: Orders Go overlay, version: 1.0.0 }
actions:
  # Force a Go type for a value property
  - target: $[?(@.declaredName=='price')]
    update: { x-go-type: "money.Money", x-go-type-import: "github.com/acme/money" }

  # Mark a part def as an aggregate and relocate its output
  - target: $[?(@.declaredName=='Order' && @['@type']=='PartDefinition')]
    update: { x-ddd-stereotype: "aggregate", x-ddd-target-dir: "internal/order/domain" }

  # Reclassify a port as a driven (outbound) port
  - target: $[?(@.declaredName=='PaymentPort')]
    update: { x-ddd-stereotype: "driven-port" }

  # Exclude internal scaffolding parts from generation entirely
  - target: $[?(@['@type']=='PartDefinition' && @.x-internal==true)]
    remove: true
```

### Actions

- `update` — recursively merged into matched nodes (inject metadata, override fields).
- `remove: true` — prune matched elements (and their memberships) before generation.
- `copy` — duplicate/move an element subtree (Overlay v1.1.0 semantics).

### Implementation

Borrow a JSONPath+merge engine (`speakeasy-api/openapi-overlay`, or libopenapi’s overlay support implementing Overlay v1.1.0) rather than reimplementing RFC 9535. Provide a friendlier **selector sugar** layer over raw JSONPath (`by name`, `by type`) since JSONPath authoring is a known friction point.

-----

## 12. Template System

- **Engine:** Go `text/template`, defaults embedded with `//go:embed templates/**/*.tmpl`.
- **Override resolution:** a built-in template is overridden by a same-named file via `output-options.user-templates` (inline string, local path, or HTTPS URL) or a `--templates <dir>` flag (same-name semantics, ent/oapi-codegen style).
- **FuncMap helpers:** `goName`, `exported`, `unexported`, `comment`, `goType`, `goTags`, `receiver`, `imports`, `zeroValue`, plus a `header` partial that emits the generated marker.
- **Default template set** (one per IR node kind, grouped by region):

```
templates/
├── domain/
│   ├── entity.go.tmpl
│   ├── aggregate.go.tmpl
│   ├── value_object.go.tmpl
│   ├── domain_event.go.tmpl
│   ├── factory.go.tmpl
│   └── domain_service.go.tmpl       (interface + scaffold impl)
├── app/
│   ├── usecase.go.tmpl              (interactor interface + scaffold impl)
│   ├── port_in.go.tmpl
│   └── port_out.go.tmpl
├── adapter/
│   ├── http_handler.go.tmpl
│   ├── repo_impl.go.tmpl
│   └── gateway_impl.go.tmpl
├── cmd/main.go.tmpl
└── _header.tmpl
```

-----

## 13. The Generator’s Own Architecture (dogfooding)

`sysgo` is itself a hexagonal Go application — the tool that emits clean architecture is built in clean architecture.

```
sysgo/
├── cmd/sysgo/main.go                  # CLI (cobra) — composition root
├── internal/
│   ├── core/                          # Domain of the generator
│   │   ├── model/                     # SysML element graph types
│   │   ├── ir/                        # DDD IR types
│   │   └── mapping/                   # SysML→IR rules (heuristics + metadata)
│   ├── app/
│   │   ├── pipeline.go                # orchestrates load→overlay→ir→render→emit
│   │   └── port/                      # driven ports (interfaces)
│   │       ├── loader.go              # ModelLoader
│   │       ├── overlay.go             # OverlayEngine
│   │       ├── renderer.go            # Renderer
│   │       └── writer.go              # FileWriter
│   ├── adapter/
│   │   ├── sysmlapi/                  # ModelLoader over REST API
│   │   ├── sysmlfile/                 # ModelLoader over JSON file
│   │   ├── overlay/                   # OverlayEngine (speakeasy/libopenapi)
│   │   ├── gotmpl/                    # Renderer (text/template + go:embed)
│   │   └── osfs/                      # FileWriter (+ gofmt/goimports)
│   └── config/                        # config parse + JSON schema validation
├── templates/                         # embedded default templates (§12)
├── schema/sysgo.schema.json
└── examples/                          # OMG sample models + golden output
```

**Future backend boundary (post-v1):** the Renderer can be replaced by a protoc-style external plugin — serialize `{IR, config}` as a `GenerateRequest`, receive `GenerateResponse{files:[{name,content}]}` over stdin/stdout or WASM. This is how non-Go targets (or third-party template packs) would plug in without forking the core.

-----

## 14. CLI / UX

```
sysgo init                      # scaffold a sysgo.yaml + empty overlay.yaml
sysgo generate [-c sysgo.yaml]  # run the pipeline
sysgo validate                  # load model + config, report mapping diagnostics, emit nothing
sysgo version
```

- `//go:generate sysgo generate -c sysgo.yaml` integration for in-repo regeneration.
- Flags mirror config keys for one-offs (`--module`, `--templates`, `--overlay`, `--out`).
- `validate` prints the resolved IR (stereotype decisions, type mappings, target paths) so users can debug mapping before writing files.

-----

## 15. Idempotent Regeneration

Adopt the goa/ent/protoc convention — **split generated from hand-written** rather than fragile in-file merge regions:

- **Generated files** carry the marker `// Code generated by sysgo; DO NOT EDIT.` (matching Go’s `^// Code generated .* DO NOT EDIT\.$` recognized by tooling) and are **always overwritten**. These are: entities, value objects, events, factories, port interfaces, use-case interfaces, DTOs.
- **Scaffold-once files** (adapter implementations, domain/use-case service impls, `cmd/main.go`) are emitted **only if absent**; never overwritten. Filled in by humans.
- **Pruning:** with `skip-prune: false`, generated files whose source element disappeared are removed (only files bearing the marker — never scaffold-once or hand-written files).
- **CI freshness:** `sysgo generate && git diff --exit-code` asserts the committed scaffold matches the model. Commit generated code to VCS (oapi-codegen’s recommendation: change impact is reviewable).

-----

## 16. Validation, Diagnostics & Errors

- **Model validation:** unresolved `@id` references, dangling memberships, cycles in composition.
- **Mapping diagnostics:** elements with no stereotype match (warn + skip), ambiguous port direction, unmapped types (error unless a default is configured).
- **Layout validation:** detect would-be outward imports implied by the mapping (e.g. a domain element configured to depend on an adapter) and fail early.
- Diagnostics are structured (element `@id`, `declaredName`, rule, severity) and surfaced by `sysgo validate`.

-----

## 17. Testing Strategy

- **Real-source pipeline:** the example model is authored in `.sysml` (`examples/order/OrderContext.sysml`) and converted to JSON by the **Pilot serializer** in CI. Generation runs from both the committed JSON and a freshly-converted one, asserting **identical generated Go** — a source-of-truth freshness check robust to the serializer’s nondeterministic element `@id`s.
- **Golden-file tests:** check the OMG/SysON sample models (vehicle, drone) into `examples/`, generate, and diff against committed golden trees; regeneration must be byte-stable and `git diff`-clean.
- **Unit tests** per stage: membership resolution, overlay application (JSONPath actions), mapping heuristics, type mapping, layout interpolation.
- **Compile test:** generated golden projects must `go build ./...` (with scaffold stubs returning `errors.New("not implemented")`).
- **Arch test:** run the emitted import-lint ruleset against the golden output to prove the Dependency Rule holds.

-----

## 18. Milestones & Issue Decomposition

> Issues are written `TASK-NN` for direct import as GitHub issues, grouped into milestones M0–M7.

### M0 — Bootstrap & Spec

- **TASK-01** Repo, `go.mod`, CI (build/test/lint), license, this `SPEC.md`.
- **TASK-02** Cobra CLI skeleton (`init`/`generate`/`validate`/`version`) with no-op pipeline.
- **TASK-03** JSON Schema for `sysgo.yaml` + `config` package with validation.

### M1 — Model Ingestion

- **TASK-04** `model.Graph`/`Element` types; membership resolution (`ownedRelationship` → `ownedRelatedElement`, `ownedElement` fast path).
- **TASK-05** `sysmlfile` loader (JSON array → graph).
- **TASK-06** `sysmlapi` loader (paged `GET …/elements`, element fetch, version pinning).
- **TASK-07** Loader conformance tests against committed sample exports.

### M2 — IR & Mapping

- **TASK-08** IR types (`Project`/`Context`/`Entity`/`Port`/`UseCase`/…).
- **TASK-09** Default heuristic mapping (part def→entity/VO, port def→port, action def→use case, package→context).
- **TASK-10** Port-direction inference (driving vs driven) + DTO extraction from item flows.
- **TASK-11** Metadata resolution (SysML metadata + `x-ddd-*`/`x-go-*` keys), precedence engine.

### M3 — Templates & Emission

- **TASK-12** `text/template` renderer + `go:embed` default set; `FuncMap`.
- **TASK-13** Domain templates (entity, aggregate, value object, event, factory, domain service).
- **TASK-14** Application templates (use case, `port/in`, `port/out`).
- **TASK-15** Adapter + `cmd` templates (http handler, repo/gateway impl, composition root).
- **TASK-16** Emitter: write + `gofmt`/`goimports`, generated marker, directory creation.

### M4 — Configuration & Layout

- **TASK-17** `layout` resolver with `{context}` interpolation → per-region dir/package.
- **TASK-18** `generate` flags (`domain`/`usecases`/`ports`/`adapters`/`events`/`tests`/`importlint`/`di`/`cmd`).
- **TASK-19** `type-mapping`, `import-mapping`, `additional-imports`, `ports.repository-in-domain`.
- **TASK-20** Emit optional arch-lint ruleset.

### M5 — Overlays & Metadata

- **TASK-21** Integrate overlay engine (`speakeasy`/libopenapi); `update`/`remove`/`copy` over model JSON.
- **TASK-22** Selector sugar (`by name`/`by type`) over raw JSONPath.
- **TASK-23** Overlay-injected `x-go-*`/`x-ddd-*` wired into mapping precedence.

### M6 — Idempotency & Scaffolding

- **TASK-24** Generated vs scaffold-once policy; never-overwrite for stubs.
- **TASK-25** Prune stale generated (marker-gated) files.
- **TASK-26** CI freshness mode + `go:generate` integration.

### M7 — Validation, Examples, Docs

- **TASK-27** `validate` command: model + mapping diagnostics, resolved-IR dump.
- **TASK-28** Golden-file harness; check in OMG/SysON sample models + golden trees.
- **TASK-29** `go build ./...` + arch-lint over golden output in CI.
- **TASK-30** User docs (getting started, config reference, overlay cookbook, template authoring).

-----

## 19. Risks & Open Questions

- **SysML v2 is still finalizing.** API JSON shape and version (OAS 3.0.1 vs 3.1) vary by implementation (e.g. SysON forces `@id == elementId`; the pilot may differ). *Mitigation:* pin to a server version; keep the Loader behind a port; cover with conformance fixtures.
- **No Go textual parser.** v1 is JSON-only by design. If textual `.sysml` ingestion is later required, generate a parser from the `antlr/grammars-v4/sysml-v2` ANTLR4 grammar (Go target) or a curated `participle`/`goyacc` subset — substantial effort (the grammar carries no normative well-formedness rules and depends on KerML library resolution).
- **JSONPath authoring friction** (acknowledged by oapi-codegen). *Mitigation:* selector sugar + a cookbook of worked overlay examples.
- **Repository-interface placement is an unresolved debate** in the canon. *Mitigation:* default to `app/port/out`, expose `ports.repository-in-domain`; document the trade-off rather than hard-coding.
- **Stereotype inference ambiguity.** Heuristics will mis-classify some part defs. *Mitigation:* `validate` surfaces every decision; metadata/overlay always overrides; warn-and-skip on no-match rather than guessing.
- **Over-generation.** Full hexagonal ceremony is wrong for CRUD-simple contexts. *Mitigation:* `generate.adapters: off|scaffold|full` and per-context opt-out; document when *not* to use the full layout.

## 20. Decision Log (to be maintained)

- **D-01** Ingest via SysML v2 **API JSON**, not textual notation (v1). — *Accepted.*
- **D-02** Output UX modeled on **oapi-codegen** (config + templates + overlays). — *Accepted.*
- **D-03** Output layout: **bounded-context-first**, four regions, dependencies inward. — *Accepted.*
- **D-04** Driven-port interfaces default to **Application** (`port/out`); configurable into Domain. — *Accepted (revisit per team).*
- **D-05** Generated/scaffold-once **file split** over in-file merge regions. — *Accepted.*
- **D-06** Examples & tests start from real **`.sysml`** textual source, transformed to API JSON by real SysML tooling (OMG Pilot serializer, `scripts/sysml2json.sh`); `sysgo` itself still ingests JSON only (textual parsing stays out of the tool). — *Accepted (revises the §1 non-goal framing).*

-----

## Appendix A — Example SysML v2 input (textual, for illustration)

```sysml
package OrderContext {
  attribute def Money { attribute amount : Real; attribute currency : String; }

  port def PaymentPort {
    out item charge  : ChargeRequest;
    in  item receipt : Receipt;
  }

  part def Order {                 // → aggregate root
    attribute id    : String;      // identity ⇒ entity/aggregate
    attribute total : Money;       // → field Total money.Money
    part lines : LineItem[*];      // → []LineItem composition
    port pay   : PaymentPort;      // → dependency on a driven port
  }

  part def LineItem { attribute sku : String; attribute qty : Integer; }

  action def PlaceOrder { in order : Order; }   // → application use case
}
```

## Appendix B — Example `sysgo.yaml`

(See §10.)

## Appendix C — Example `overlay.yaml`

(See §11.)

## Appendix D — Resulting tree

(See §9.2.)

# OPENAPI — sysgo Contracts Spec (OpenAPI profile)

**Version:** c0.1
**Status:** OpenAPI profile decided end-to-end. GraphQL and protobuf profiles
deferred until a consumer needs them.
**Parent:** `OVERVIEW.md`. **Depends on:** `ENGINE.md`.

---

## 1. Purpose and scope

Generates API contracts from the model. Per framework §7 the contract profiles
are **per-protocol and separate**; this spec covers the **OpenAPI** profile
first. GraphQL and protobuf profiles are deferred (§10) — the pattern here
generalizes to them.

Direction is **model-first** (framework): the SysML model is the source of truth;
sysgo emits the OpenAPI document, and `oapi-codegen` (imported as a library, §7)
turns it into Go. Consumers: Epos, Codiq (and later Guix, mcp-anything). The
emitted `openapi.yaml` is a first-class, publishable artifact.

---

## 2. Pipeline (in-process, staged)

oapi-codegen is used as a **library**, not a CLI (v2 API). One in-process,
ordered pipeline per API:

1. **config** — build a fixed-default `codegen.Configuration`
   (`gin-server`, `strict-server`, `models`, `client` on; OpenAPI 3.1). Optionally
   serialized to `oapi-codegen.yaml` for reproducibility/manual runs.
2. **openapi** — build the OpenAPI 3.1 document from the model and serialize it to
   `openapi.yaml` (the publishable artifact).
3. **generate** — pass the spec to `codegen.Generate(spec, config)`; write the
   generated `ServerInterface`, models, and gin wiring.
4. **adapter** — emit (jennifer) the struct implementing `ServerInterface`, each
   method binding request → domain action → response (§4).

The domain use-case body is **not** generated — it is the hexagonal core, in a
separate hand-written file.

---

## 3. Representation — the REST metadata profile

An HTTP API surface is modeled with native SysML constructs + a REST metadata
profile (framework §6). Boundary: **fully explicit** — `@REST` carries the full
contract per operation (not convention-derived).

```
package RESTProfile {
    metadata def Api {                          // API-level
        attribute basePath : String;
        attribute version  : String;
        attribute security : String[*];         // default security schemes
    }
    metadata def REST {                         // operation-level
        attribute path            : String;
        attribute method          : HttpMethod; // GET/POST/PUT/PATCH/DELETE/...
        attribute successStatus   : Integer;    // e.g. 200, 201, 204
        attribute security        : String[*];  // per-op override of Api.security
        attribute requestContentType  : String; // default application/json
        attribute responseContentType : String; // default application/json
    }
    metadata def ErrorModel { attribute schemaRef : String; } // override; see §6
}
```

Request/response **bodies are not restated in metadata** — they are the operation
action's `in`/`out` parameters, typed by `item def`s (§4, §5).

---

## 4. Transport ↔ domain binding

Per the locked framework binding (transport is a driving adapter over
transport-free domain actions):

- A transport **operation** is an `action` carrying `@REST`, whose body
  `perform`s the target domain `action def`. The call graph transport→use-case is
  real model structure.
- The operation's `in`/`out` parameters (typed by `item def`s) are the request and
  response bodies; they map to the domain action's parameters.
- The generated **adapter** (§2 step 4) implements the oapi-codegen
  `ServerInterface` method: decode request → call the domain action → encode
  response / map error. One use case may be exposed over multiple protocols and
  also triggered by Temporal without duplication, since the domain action is
  transport-free.

---

## 5. `item def` → JSON Schema mapping (OpenAPI 3.1)

3.1 target = JSON Schema 2020-12 alignment. Rules:

| SysML | JSON Schema (3.1) |
| --- | --- |
| `String` | `type: string` |
| `Integer` | `type: integer` (format by width) |
| `Real` | `type: number` |
| `Boolean` | `type: boolean` |
| `Duration` | `type: string, format: duration` |
| enumeration def | string schema with `enum` (or `oneOf`+`const`) |
| optional attribute | omitted from `required` |
| nullable attribute | 3.1 idiom `type: [T, "null"]` |

**Specialization → flattened (locked).** A specialized `item def` emits a
**self-contained** schema with all inherited attributes inlined; **no `allOf`.**
This gives the cleanest oapi-codegen Go output (plain structs, no embedded-struct
/ `required` / `additionalProperties` edge cases) and maximal client portability,
at the cost of a larger document and a contract that does not mirror the model's
inheritance.

---

## 6. Error model — RFC 9457 default + override

- **Default:** RFC 9457 Problem Details (`application/problem+json`;
  `type`/`title`/`status`/`detail`/`instance` + extensions). The profile ships a
  standard problem-details schema; operations reference it by default.
- **Override:** `@ErrorModel` on an operation/API supplies a custom error schema
  where needed. Precedence: override present → custom; else Problem Details.

Interoperable standard for published contracts, with a bespoke escape hatch.

---

## 7. Generation details

- **oapi-codegen as a library (v2).** `codegen.Generate(spec, config)` consumes a
  parsed spec + a `codegen.Configuration`. The config struct is oapi-codegen's own
  config schema, satisfying "same config as oapi-codegen".
- **Config defaults (fixed scaffolding, parallel to CI/F9):** `gin-server`,
  `strict-server`, `models`, `client` on, OpenAPI 3.1. Server framework and
  client on/off live in config, not the model.
- **3.1 idioms** used: `type: [T,"null"]` nullability, `oneOf`+`const` enums.
- **Adapter** emitted with **jennifer** (framework §5a); `openapi.yaml` and any
  config YAML emitted as text (templates).
- **Generated vs hand-written:** generated (wholesale) = `openapi.yaml`, config,
  oapi-codegen output (interface/models/gin), and the adapter. Hand-written = the
  domain use-case body (separate file), and any `@ErrorModel` custom schemas'
  domain meaning.

---

## 8. Deployment

Framework §6 applies. The `openapi.yaml` is published as a first-class artifact;
projects exposing a docs/preview site follow the branch-mode + PR-preview rule.

---

## 9. Decision log

- **C1.** OpenAPI profile first; GraphQL/protobuf deferred (separate per-protocol
  profiles).
- **C2.** Model-first; sysgo emits `openapi.yaml`, oapi-codegen emits Go.
- **C3.** `@REST` fully explicit (path, method, status, security, content types).
- **C4.** Transport operation `perform`s the domain action; bodies = action
  in/out `item def`s.
- **C5.** OpenAPI **3.1**.
- **C6.** Error model: RFC 9457 default + `@ErrorModel` override.
- **C7.** Server framework = **gin**, client **on** — via oapi-codegen config
  (fixed default), not the model.
- **C8.** Adapter generated (jennifer) implementing `ServerInterface`; domain body
  hand-written.
- **C9.** oapi-codegen used as an **imported library** (v2), in-process staged
  pipeline.
- **C10.** `item def` specialization → **flattened** self-contained schemas (no
  `allOf`).

---

## 10. Open / risk items

- **Dependency pins/risks:** pin oapi-codegen (v2) and kin-openapi; kin-openapi is
  pre-v1 (breaking-change risk) and 3.0-centric (3.1 via version-aware handling —
  verify 3.1 idioms round-trip); plan for the oapi-codegen v2→v3 API migration.
- **Spec construction flow:** build the OpenAPI doc as an in-memory kin-openapi
  `openapi3.T` and serialize (avoids a round-trip but exercises the 3.0-centric
  model) **vs** emit `openapi.yaml` as text and load it (sidesteps model
  awkwardness, adds a serialize→parse round-trip) — to be chosen in
  implementation.
- **GraphQL profile** (gqlgen) and **protobuf/gRPC profile** (buf + connect-go) —
  deferred; bring forward when a consumer needs them (e.g. Codiq internal gRPC).
- oapi-codegen 3.1 support is experimental — pin to a known-good version/commit.

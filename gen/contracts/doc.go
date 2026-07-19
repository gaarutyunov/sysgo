// Package contracts is the contracts (API) generator that consumes the sysgo
// engine's resolved model. This first slice implements the item-definition →
// JSON Schema 2020-12 mapping (specs/OPENAPI.md §5), the foundation the OpenAPI
// document builder and downstream oapi-codegen pipeline build on.
//
// A SysML item/attribute definition maps to a JSON object schema: each attribute
// becomes a property typed by its resolved feature-typing target. Scalar library
// types (ScalarValues::String/Integer/Real/Boolean/Natural) map to JSON Schema
// scalar types; an attribute typed by another definition inlines that
// definition's object schema recursively.
//
// Specialization is flattened (decision C10): a specialized definition emits a
// self-contained schema with every inherited attribute inlined — no allOf — for
// the cleanest downstream Go output.
package contracts

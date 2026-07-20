package contracts

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// ToOpenAPI3 converts the in-memory Document into a kin-openapi *openapi3.T,
// constructing the OpenAPI 3.1 model directly as Go structs — no YAML round-trip.
// This is what feeds oapi-codegen (see generate); Document.YAML() is retained
// only for the optional reference document (swagger/redocly, etc.).
func (d *Document) ToOpenAPI3() *openapi3.T {
	t := &openapi3.T{
		OpenAPI: d.OpenAPI,
		Info:    &openapi3.Info{Title: d.Info.Title, Version: d.Info.Version},
		Paths:   openapi3.NewPaths(),
	}
	if d.Components != nil && len(d.Components.Schemas) > 0 {
		schemas := openapi3.Schemas{}
		for name, s := range d.Components.Schemas {
			schemas[name] = schemaToT(s)
		}
		t.Components = &openapi3.Components{Schemas: schemas}
	}
	for path, item := range d.Paths {
		t.Paths.Set(path, &openapi3.PathItem{
			Get:    operationToT(item.Get),
			Put:    operationToT(item.Put),
			Post:   operationToT(item.Post),
			Patch:  operationToT(item.Patch),
			Delete: operationToT(item.Delete),
		})
	}
	return t
}

func operationToT(op *Operation) *openapi3.Operation {
	if op == nil {
		return nil
	}
	responses := openapi3.NewResponses()
	for code, r := range op.Responses {
		responses.Set(code, responseToT(r))
	}
	out := &openapi3.Operation{OperationID: op.OperationID, Responses: responses}
	if op.RequestBody != nil {
		out.RequestBody = requestBodyToT(op.RequestBody)
	}
	return out
}

func requestBodyToT(rb *RequestBody) *openapi3.RequestBodyRef {
	body := &openapi3.RequestBody{Required: rb.Required}
	if len(rb.Content) > 0 {
		content := openapi3.Content{}
		for mediaType, m := range rb.Content {
			content[mediaType] = &openapi3.MediaType{Schema: schemaToT(m.Schema)}
		}
		body.Content = content
	}
	return &openapi3.RequestBodyRef{Value: body}
}

func responseToT(r *Response) *openapi3.ResponseRef {
	desc := r.Description
	out := &openapi3.Response{Description: &desc}
	if len(r.Content) > 0 {
		content := openapi3.Content{}
		for mediaType, m := range r.Content {
			content[mediaType] = &openapi3.MediaType{Schema: schemaToT(m.Schema)}
		}
		out.Content = content
	}
	return &openapi3.ResponseRef{Value: out}
}

// schemaToT maps the internal Schema (a JSON-Schema subset) to a kin-openapi
// *openapi3.SchemaRef. A Schema carrying a $ref becomes a reference; otherwise an
// inline schema is built recursively.
func schemaToT(s *Schema) *openapi3.SchemaRef {
	if s == nil {
		return nil
	}
	if s.Ref != "" {
		return openapi3.NewSchemaRef(s.Ref, nil)
	}
	out := &openapi3.Schema{Format: s.Format}
	if s.Type != "" {
		types := openapi3.Types{s.Type}
		out.Type = &types
	}
	if len(s.Enum) > 0 {
		out.Enum = make([]any, len(s.Enum))
		for i, e := range s.Enum {
			out.Enum[i] = e
		}
	}
	if s.Items != nil {
		out.Items = schemaToT(s.Items)
	}
	if len(s.Properties) > 0 {
		props := openapi3.Schemas{}
		for k, v := range s.Properties {
			props[k] = schemaToT(v)
		}
		out.Properties = props
	}
	if len(s.Required) > 0 {
		out.Required = append([]string(nil), s.Required...)
	}
	return openapi3.NewSchemaRef("", out)
}

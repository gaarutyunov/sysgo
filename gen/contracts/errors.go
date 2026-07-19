package contracts

import (
	"strings"

	"github.com/gaarutyunov/sysgo/engine"
)

// ProblemDetailsSchemaName is the component name of the default RFC 9457 error
// schema.
const ProblemDetailsSchemaName = "ProblemDetails"

const (
	problemDetailsRef     = "#/components/schemas/" + ProblemDetailsSchemaName
	problemJSONContentTyp = "application/problem+json"
)

// problemDetailsSchema returns the RFC 9457 Problem Details object schema
// (type/title/status/detail/instance).
func problemDetailsSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"type":     {Type: "string", Format: "uri"},
			"title":    {Type: "string"},
			"status":   {Type: "integer"},
			"detail":   {Type: "string"},
			"instance": {Type: "string", Format: "uri"},
		},
	}
}

// errorResponse builds the default error response referencing schemaRef via
// application/problem+json.
func errorResponse(schemaRef string) *Response {
	return &Response{
		Description: "error",
		Content: map[string]*MediaType{
			problemJSONContentTyp: {Schema: &Schema{Ref: schemaRef}},
		},
	}
}

// errorSchemaRef resolves an operation's error-schema $ref: an @ErrorModel
// override (its schemaRef, taken verbatim or expanded to a component ref) takes
// precedence; otherwise the default RFC 9457 Problem Details. The second return
// reports whether the default Problem Details schema is used (so the caller can
// add it to components).
func errorSchemaRef(e engine.Element) (string, bool) {
	if meta, ok := e.Metadata("ErrorModel"); ok {
		if v, ok := meta.Value("schemaRef"); ok {
			if ref := unquote(v); ref != "" {
				if !strings.HasPrefix(ref, "#/") {
					ref = "#/components/schemas/" + ref
				}
				return ref, false
			}
		}
	}
	return problemDetailsRef, true
}

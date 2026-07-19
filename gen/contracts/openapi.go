package contracts

import (
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gaarutyunov/sysgo/engine"
)

// Document is a minimal OpenAPI 3.1 document model — enough to emit info, paths,
// and component schemas deterministically. Request/response body binding (which
// needs parameter direction) is a later slice.
type Document struct {
	OpenAPI    string               `yaml:"openapi"`
	Info       Info                 `yaml:"info"`
	Paths      map[string]*PathItem `yaml:"paths,omitempty"`
	Components *Components          `yaml:"components,omitempty"`
}

// Info is the OpenAPI info object.
type Info struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

// PathItem holds the operations for one path.
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Patch  *Operation `yaml:"patch,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

func (p *PathItem) set(method string, op *Operation) {
	switch strings.ToUpper(method) {
	case "GET":
		p.Get = op
	case "PUT":
		p.Put = op
	case "POST":
		p.Post = op
	case "PATCH":
		p.Patch = op
	case "DELETE":
		p.Delete = op
	default:
		p.Get = op // unknown method → default to GET slot
	}
}

// Operation is one HTTP operation.
type Operation struct {
	OperationID string               `yaml:"operationId,omitempty"`
	RequestBody *RequestBody         `yaml:"requestBody,omitempty"`
	Responses   map[string]*Response `yaml:"responses,omitempty"`
}

// Response is one HTTP response.
type Response struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content,omitempty"`
}

// MediaType is a response/request content entry for one media type.
type MediaType struct {
	Schema *Schema `yaml:"schema,omitempty"`
}

// RequestBody is an OpenAPI request body object.
type RequestBody struct {
	Required bool                  `yaml:"required,omitempty"`
	Content  map[string]*MediaType `yaml:"content,omitempty"`
}

// Components holds the reusable component schemas.
type Components struct {
	Schemas map[string]*Schema `yaml:"schemas,omitempty"`
}

// BuildDocument assembles an OpenAPI 3.1 document from a resolved model:
// components/schemas from definitions that have attributes, and paths from
// @REST-annotated operations. Each operation gets a default error response
// (RFC 9457 Problem Details, or an @ErrorModel override).
func BuildDocument(m *engine.Model) *Document {
	doc := &Document{OpenAPI: "3.1.0", Info: Info{Title: "API", Version: "1.0.0"}}
	schemas := map[string]*Schema{}
	paths := map[string]*PathItem{}
	needProblemDetails := false

	visit := func(e engine.Element) {
		if meta, ok := e.Metadata("REST"); ok {
			if addOperation(paths, e, meta) {
				needProblemDetails = true
			}
			return // an operation is not itself a data schema
		}
		if isSchemaDefinition(e) {
			schemas[componentName(e.QualifiedName())] = SchemaFor(e)
		}
	}
	// Walk only user packages; the bundled standard library and metadata
	// profiles are not part of the emitted contract.
	for _, top := range m.Root().Children() {
		if isBundledPackage(top.Name()) {
			continue
		}
		walk(top, visit)
	}

	if needProblemDetails {
		schemas[ProblemDetailsSchemaName] = problemDetailsSchema()
	}
	if len(schemas) > 0 {
		doc.Components = &Components{Schemas: schemas}
	}
	if len(paths) > 0 {
		doc.Paths = paths
	}
	return doc
}

// YAML serializes the document to deterministic OpenAPI YAML.
func (d *Document) YAML() string {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	// Encode error is impossible for this in-memory value graph.
	_ = enc.Encode(d)
	_ = enc.Close()
	return b.String()
}

// isBundledPackage reports whether name is an engine-bundled top-level package
// (standard library or metadata profile) that must be excluded from generated
// contracts.
func isBundledPackage(name string) bool {
	switch name {
	case "ScalarValues", "Base", "RESTProfile", "TemporalProfile":
		return true
	default:
		return false
	}
}

// walk visits e and all its descendants in pre-order.
func walk(e engine.Element, fn func(engine.Element)) {
	fn(e)
	for _, c := range e.Children() {
		walk(c, fn)
	}
}

// isSchemaDefinition reports whether e is a definition that should become a
// component schema: a definition with at least one attribute and no @REST
// annotation (operations are excluded).
func isSchemaDefinition(e engine.Element) bool {
	if e.Kind() != engine.ElementDefinition {
		return false
	}
	if _, ok := e.Metadata("REST"); ok {
		return false
	}
	for _, c := range e.Children() {
		if c.Kind() == engine.ElementUsage {
			return true
		}
	}
	return false
}

// addOperation adds an operation for an @REST element and returns whether it
// used the default Problem Details error schema (vs. an @ErrorModel override).
func addOperation(paths map[string]*PathItem, e engine.Element, meta engine.Metadata) bool {
	path := unquote(valueOr(meta, "path", "/"+e.Name()))
	method := unquote(valueOr(meta, "method", "GET"))
	status := unquote(valueOr(meta, "successStatus", "200"))

	item := paths[path]
	if item == nil {
		item = &PathItem{}
		paths[path] = item
	}
	ref, usedDefault := errorSchemaRef(e)
	success := &Response{Description: e.Name() + " succeeded"}
	if mt := bodySchemaFor(e, "out"); mt != nil {
		success.Content = map[string]*MediaType{jsonMediaType: mt}
	}
	op := &Operation{
		OperationID: e.Name(),
		Responses: map[string]*Response{
			status:    success,
			"default": errorResponse(ref),
		},
	}
	if mt := bodySchemaFor(e, "in"); mt != nil {
		op.RequestBody = &RequestBody{Required: true, Content: map[string]*MediaType{jsonMediaType: mt}}
	}
	item.set(method, op)
	return usedDefault
}

func valueOr(meta engine.Metadata, key, fallback string) string {
	if v, ok := meta.Value(key); ok && v != "" {
		return v
	}
	return fallback
}

// unquote strips surrounding quotes from a string-literal metadata value; a
// non-quoted value (e.g. a number) is returned as-is.
func unquote(s string) string {
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return s
}

const jsonMediaType = "application/json"

// bodySchemaFor builds the media-type schema for an operation's request (dir
// "in") or response (dir "out") body from the operation action's directed
// parameters typed by item definitions. A single directed parameter refs its
// type's component schema; multiple are wrapped in an object with one property
// per parameter. Returns nil when the operation has no such parameter.
func bodySchemaFor(op engine.Element, dir string) *MediaType {
	var params []engine.Element
	for _, c := range op.Children() {
		if c.Kind() != engine.ElementUsage || c.Direction() != dir {
			continue
		}
		if t, ok := paramSchemaType(c); ok {
			_ = t
			params = append(params, c)
		}
	}
	switch len(params) {
	case 0:
		return nil
	case 1:
		t, _ := paramSchemaType(params[0])
		return &MediaType{Schema: refSchema(t)}
	default:
		obj := &Schema{Type: "object", Properties: map[string]*Schema{}}
		for _, p := range params {
			t, _ := paramSchemaType(p)
			obj.Properties[p.Name()] = refSchema(t)
			obj.Required = append(obj.Required, p.Name())
		}
		sort.Strings(obj.Required)
		return &MediaType{Schema: obj}
	}
}

// paramSchemaType returns a directed parameter's item-definition type, if it is
// typed by a definition (not a scalar) that becomes a component schema.
func paramSchemaType(p engine.Element) (engine.Element, bool) {
	t, ok := attributeType(p)
	if !ok || isScalar(t.QualifiedName()) {
		return engine.Element{}, false
	}
	if t.Kind() != engine.ElementDefinition {
		return engine.Element{}, false
	}
	return t, true
}

// refSchema builds a $ref schema pointing at a definition's component schema.
func refSchema(def engine.Element) *Schema {
	return &Schema{Ref: "#/components/schemas/" + componentName(def.QualifiedName())}
}

// componentName turns a qualified name into an OpenAPI component key ("::" is
// not allowed in component names).
func componentName(qn string) string {
	return strings.ReplaceAll(qn, "::", ".")
}

// SchemaNames returns the component schema names in sorted order (helper for
// callers/tests).
func (d *Document) SchemaNames() []string {
	if d.Components == nil {
		return nil
	}
	out := make([]string, 0, len(d.Components.Schemas))
	for k := range d.Components.Schemas {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

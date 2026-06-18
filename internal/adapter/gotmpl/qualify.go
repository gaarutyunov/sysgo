package gotmpl

import (
	"strings"
	"unicode"

	"github.com/gaarutyunov/sysgo/internal/core/ir"
)

// The qualify* helpers return shallow copies of IR nodes whose Go type strings
// have been package-qualified for the consuming file, while recording imports
// on q. The originals are never mutated.

func qualifyFields(in []*ir.Field, q *qualifier) []*ir.Field {
	out := make([]*ir.Field, len(in))
	for i, f := range in {
		c := *f
		c.GoType = q.qualify(f.GoType)
		out[i] = &c
	}
	return out
}

func qualifyMethods(in []*ir.Method, q *qualifier) []*ir.Method {
	out := make([]*ir.Method, len(in))
	for i, m := range in {
		c := *m
		c.Params = qualifyParams(m.Params, q)
		c.Results = qualifyParams(m.Results, q)
		out[i] = &c
	}
	return out
}

func qualifyParams(in []*ir.Param, q *qualifier) []*ir.Param {
	out := make([]*ir.Param, len(in))
	for i, p := range in {
		c := *p
		c.GoType = q.qualify(p.GoType)
		out[i] = &c
	}
	return out
}

func qualifyEntity(e *ir.Entity, q *qualifier) *ir.Entity {
	c := *e
	c.Fields = qualifyFields(e.Fields, q)
	c.Methods = qualifyMethods(e.Methods, q)
	return &c
}

func qualifyVO(v *ir.ValueObject, q *qualifier) *ir.ValueObject {
	c := *v
	c.Fields = qualifyFields(v.Fields, q)
	return &c
}

func qualifyEvent(e *ir.DomainEvent, q *qualifier) *ir.DomainEvent {
	c := *e
	c.Fields = qualifyFields(e.Fields, q)
	return &c
}

func qualifyService(s *ir.DomainService, q *qualifier) *ir.DomainService {
	c := *s
	c.Methods = qualifyMethods(s.Methods, q)
	return &c
}

func qualifyPort(p *ir.Port, q *qualifier) *ir.Port {
	c := *p
	c.Methods = qualifyMethods(p.Methods, q)
	return &c
}

func qualifyDTO(d *ir.DTO, q *qualifier) *ir.DTO {
	c := *d
	c.Fields = qualifyFields(d.Fields, q)
	return &c
}

func qualifyUseCase(uc *ir.UseCase, q *qualifier) *ir.UseCase {
	c := *uc
	c.Input = qualifyDTO(uc.Input, q)
	c.Output = qualifyDTO(uc.Output, q)
	c.Port = qualifyPort(uc.Port, q)
	return &c
}

// snake converts an exported Go name into snake_case for file names.
func snake(name string) string {
	var b strings.Builder
	var prev rune
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(prev) || (prev != 0 && unicode.IsLetter(prev) && i+1 < len(name) && unicode.IsLower(rune(name[i+1])))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
		prev = r
	}
	return b.String()
}

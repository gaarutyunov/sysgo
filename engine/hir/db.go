package hir

import (
	"context"

	"github.com/gaarutyunov/sysgo/engine/salsa"
)

// report is the cheap, value-comparable output of the analysis query — small
// enough that salsa's backdating (an edit that does not change diagnostics is
// not propagated) stays inexpensive.
type report struct {
	diagnostics   []Diagnostic
	names         []ResolvedRef
	relationships []RelRef
}

// Db is the incremental façade over the HIR pipeline. Each source file is a
// durable salsa input keyed by an arbitrary file key; analysis is a tracked
// query, so re-setting one file re-analyzes only that file, and an edit that
// leaves diagnostics unchanged does not disturb downstream consumers.
type Db struct {
	sdb      *salsa.Db
	source   *salsa.Input[string, string]
	analysis *salsa.Query[string, report]
}

// NewDb returns an empty incremental HIR database.
func NewDb() *Db {
	d := &Db{
		sdb:    salsa.New(),
		source: salsa.NewInput[string, string]("hir.source"),
	}
	d.analysis = salsa.NewQuery("hir.analysis", func(c *salsa.Ctx, key string) report {
		src := d.source.Get(c, key)
		r := Analyze(src)
		return report{diagnostics: r.Diagnostics, names: r.Names, relationships: r.Relationships}
	})
	return d
}

// SetSource sets (or replaces) the source text for a file key.
func (d *Db) SetSource(key, src string) { d.source.Set(d.sdb, key, src) }

// Diagnostics returns the resolution diagnostics for a file key.
func (d *Db) Diagnostics(key string) []Diagnostic {
	var out []Diagnostic
	_ = d.sdb.Read(context.Background(), func(c *salsa.Ctx) {
		out = d.analysis.Get(c, key).diagnostics
	})
	return out
}

// Names returns the resolved references (import targets) for a file key.
func (d *Db) Names(key string) []ResolvedRef {
	var out []ResolvedRef
	_ = d.sdb.Read(context.Background(), func(c *salsa.Ctx) {
		out = d.analysis.Get(c, key).names
	})
	return out
}

// Relationships returns the resolved relationship references for a file key.
func (d *Db) Relationships(key string) []RelRef {
	var out []RelRef
	_ = d.sdb.Read(context.Background(), func(c *salsa.Ctx) {
		out = d.analysis.Get(c, key).relationships
	})
	return out
}

// Package engine is the public, in-process Go API of the sysgo engine
// (specs/ENGINE.md §9). It is the surface the generators, codiq, and Epos import
// to traverse a fully resolved SysML v2 / KerML model — type-safe and with zero
// marshalling. No serialized model is part of this surface.
//
// Typical use:
//
//	m := engine.New().
//		AddFile("order.sysml", src).
//		Build()
//	if len(m.Diagnostics()) != 0 { /* handle unresolved names */ }
//	for _, pkg := range m.Root().Children() {
//		for _, member := range pkg.Children() {
//			for _, rel := range member.Relationships() {
//				if t, ok := rel.Target(); ok {
//					// member <rel.Kind()> t
//				}
//			}
//		}
//	}
//
// The standard library is loaded automatically, so library names (Base::…,
// ScalarValues::…) resolve without any extra setup. Elements are lightweight
// views over the resolved model; they own no data and are cheap to pass around.
package engine

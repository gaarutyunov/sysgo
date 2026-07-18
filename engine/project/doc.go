// Package project is Layer 4 of the sysgo engine (specs/ENGINE.md §8): workspace
// and standard-library loading.
//
// A [Workspace] holds the user's source files together with the embedded SysML
// v2 standard library, and resolves the whole set as one namespace via package
// hir. Because most real models specialize library elements (Base::…,
// ScalarValues::…), the standard library is part of the engine, not an external
// dependency — it is embedded with go:embed and loaded into every workspace.
//
// The bundled library under stdlib/ is a curated, parser-compatible subset of
// the OMG SysML v2 release (MIT-licensed; see stdlib file headers). The loader
// reads every embedded .sysml file, so growing the bundle toward the full
// library is a drop-in change requiring no code edits.
package project

package cst

// RawKind is the compact syntax-kind tag stored on every green node and token.
// It is a u16-equivalent, kept small so green nodes stay cache-dense in the
// arena (ENGINE §5).
//
// The cst layer treats RawKind as opaque: it never interprets the value. The
// concrete KerML/SysML kind set and the raw-tag ↔ typed-kind mapping are the
// parser layer's responsibility (ENGINE §5b). Callers define their own kind
// constants and, for debugging, may supply a [KindNamer] to render them.
type RawKind uint16

// KindNamer renders a [RawKind] as a human-readable name. It is used only for
// debug output ([Print]); an unknown kind may return "".
type KindNamer func(RawKind) string

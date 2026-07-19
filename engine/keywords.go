package engine

// Keywords returns the element's leading declaration keywords (modifiers, kind
// keyword, and optional "def") in source order.
func (e Element) Keywords() []string { return e.sym.Keywords }

// Direction returns the element's parameter direction ("in", "out", or "inout")
// for a directed usage, or "" if none.
func (e Element) Direction() string {
	for _, k := range e.sym.Keywords {
		switch k {
		case "in", "out", "inout":
			return k
		}
	}
	return ""
}

// DeclarationKeyword returns the element's SysML kind keyword — the noun that
// names its kind, e.g. "part", "attribute", "item", "action", "classifier" —
// skipping modifiers and "def". It returns "" if none is present.
func (e Element) DeclarationKeyword() string {
	for _, k := range e.sym.Keywords {
		if k == "def" || isModifierKeyword(k) {
			continue
		}
		return k
	}
	return ""
}

func isModifierKeyword(s string) bool {
	switch s {
	case "in", "out", "inout", "abstract", "ref", "readonly", "derived", "end",
		"composite", "portion", "variation", "individual", "snapshot", "timeslice":
		return true
	default:
		return false
	}
}

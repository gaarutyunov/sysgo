package engine

// ControlKind returns this element's control-node kind ("fork", "join",
// "merge", or "decide"), or "" if the element is not a control node.
func (e Element) ControlKind() string { return e.sym.ControlKind }

// IsControlNode reports whether the element is a control node.
func (e Element) IsControlNode() bool { return e.sym.ControlKind != "" }

// ControlNodes returns the element's directly declared control nodes, in order.
func (e Element) ControlNodes() []Element {
	var out []Element
	for _, c := range e.Children() {
		if c.IsControlNode() {
			out = append(out, c)
		}
	}
	return out
}

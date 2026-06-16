// Package model holds the in-memory representation of a SysML v2 element
// graph. It is the output of the Loader stage and the input to the IR builder.
//
// SysML v2 serializes elements as JSON-LD-ish objects keyed by "@id"/"@type".
// Containment is indirect: an Element owns a Membership via "ownedRelationship",
// and the Membership points at the contained Element via "ownedRelatedElement".
// Where a derived "ownedElement" array is present we use it as a fast path.
package model

// Element is a single SysML v2 node, identified by its "@id".
type Element struct {
	ID           string
	Type         string
	DeclaredName string

	// Raw is the full decoded JSON for the element. Overlays operate on this
	// map, and the mapper reads x-go-*/x-ddd-* hints from it.
	Raw map[string]any

	// Owned holds the resolved child elements (via memberships).
	Owned []*Element
	// Owner points back at the containing element (nil for roots).
	Owner *Element
}

// Graph is the resolved element graph keyed by "@id".
type Graph struct {
	Elements map[string]*Element
	Roots    []*Element
}

// QualifiedName returns the declared name; callers that need a fully qualified
// name should walk Owner. It is provided for convenience in mapping rules.
func (e *Element) QualifiedName() string {
	if e == nil {
		return ""
	}
	if e.Owner == nil || e.Owner.DeclaredName == "" {
		return e.DeclaredName
	}
	return e.Owner.QualifiedName() + "::" + e.DeclaredName
}

// Attr returns the raw attribute value for a key, if present.
func (e *Element) Attr(key string) (any, bool) {
	if e == nil || e.Raw == nil {
		return nil, false
	}
	v, ok := e.Raw[key]
	return v, ok
}

// StringAttr returns a string attribute, or "" if absent or not a string.
func (e *Element) StringAttr(key string) string {
	v, ok := e.Attr(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// BoolAttr returns a boolean attribute, or false if absent/not a bool.
func (e *Element) BoolAttr(key string) bool {
	v, ok := e.Attr(key)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

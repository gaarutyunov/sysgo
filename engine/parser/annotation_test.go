package parser

import "testing"

func TestAnnotationBodyStructure(t *testing.T) {
	src := "@REST {\n\tpath = \"/orders\";\n\tmethod = POST;\n} part def Order;"
	tree := roundTrip(t, src)
	k := nodeKinds(tree.Root())

	// The Def carries an Annotation whose Body holds two assignment usages.
	if countKind(k, "Annotation") != 1 {
		t.Fatalf("Annotation count = %d, want 1 (%v)", countKind(k, "Annotation"), k)
	}
	if countKind(k, "Def") != 1 {
		t.Errorf("Def count = %d, want 1", countKind(k, "Def"))
	}
	// Two FeatureValue nodes: path=... and method=...
	if countKind(k, "FeatureValue") != 2 {
		t.Errorf("FeatureValue count = %d, want 2", countKind(k, "FeatureValue"))
	}
}

func TestAnnotationBodyRoundTrips(t *testing.T) {
	for _, src := range []string{
		"@REST { path = \"/x\"; } part def P;",
		"@Api { basePath = \"/v1\"; version = \"1.0\"; }\npackage M {}",
		"#command action a;", // bodyless annotation still fine
		"@Typed { x : Real = 1.5; } part def Q;",
	} {
		roundTrip(t, src)
	}
}

func TestBodylessAnnotationUnaffected(t *testing.T) {
	tree := roundTrip(t, "@Approved part def X;")
	k := nodeKinds(tree.Root())
	if countKind(k, "Annotation") != 1 {
		t.Errorf("Annotation count = %d, want 1", countKind(k, "Annotation"))
	}
	// No body, so no assignment usages under the annotation.
	if countKind(k, "FeatureValue") != 0 {
		t.Errorf("FeatureValue count = %d, want 0", countKind(k, "FeatureValue"))
	}
}

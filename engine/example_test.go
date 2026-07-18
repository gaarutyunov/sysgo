package engine_test

import (
	"fmt"

	"github.com/gaarutyunov/sysgo/engine"
)

// Example builds a small model and walks a resolved relationship through the
// public API.
func Example() {
	m := engine.New().
		AddFile("m.sysml", "package M {\n\tpart def Vehicle;\n\tpart def Car :> Vehicle;\n}").
		Build()

	car, _ := m.Lookup("M::Car")
	for _, rel := range car.Relationships() {
		if target, ok := rel.Target(); ok {
			fmt.Printf("%s %s %s\n", car.QualifiedName(), rel.Kind(), target.QualifiedName())
		}
	}
	// Output:
	// M::Car specializes M::Vehicle
}

// Command sysgo generates a Domain-Driven, hexagonal Go project scaffold from a
// SysML v2 model. See SPEC.md for the full design.
package main

import (
	"os"

	"github.com/gaarutyunov/sysgo/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}

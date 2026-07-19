// Command temporal is a self-contained, runnable example of the sysgo Temporal
// generator. The SysML model in model.sysml is turned into the Go code under
// ./orders by `sysgo gen temporal`; this package supplies the hand-written
// activity implementations behind the generated Activities port and a worker
// main that runs them against a real Temporal server.
//
// Regenerate the ./orders package after editing model.sysml with:
//
//	go generate ./...
//
//go:generate go tool sysgo gen temporal model.sysml --out orders --package orders
package main

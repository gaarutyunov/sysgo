// Package temporal is the Temporal Go generator that consumes the sysgo engine's
// resolved model (specs/TEMPORAL.md). This first slice classifies the model:
// discovering @Workflow and @Activity actions and reading their Temporal
// configuration (task queue, retry policy, timeouts, idempotency) via the engine
// metadata API.
//
// The TemporalProfile metadata definitions (@Workflow/@Activity/@Signal/… and a
// Duration datatype) are bundled with the engine and loaded into every
// workspace, so the annotations resolve in-model without per-project vendoring.
//
// Code emission (activity interfaces, workflow functions, workers, schedules)
// is delivered by later slices on top of this classification.
package temporal

package temporal

// Blank imports keep the Temporal SDK packages the generated worker/workflow
// code uses in the module graph and cache, so the compile tests build offline.
import (
	_ "go.temporal.io/sdk/client"
	_ "go.temporal.io/sdk/worker"
	_ "go.temporal.io/sdk/workflow"
)

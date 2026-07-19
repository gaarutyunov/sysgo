// Temporal metadata profile (specs/TEMPORAL.md §2). Bundled with the engine and
// loaded into every workspace so @Workflow / @Activity / @Signal / … annotations
// resolve in-model (T1/T2). Kept parser-compatible.
package TemporalProfile {
	import ScalarValues::*;

	datatype Duration :> String;

	metadata def Workflow {
		attribute id : String;
		attribute taskQueue : String;
	}

	metadata def Activity {
		attribute taskQueue : String;
	}

	metadata def Signal {
		attribute name : String;
	}

	metadata def Query {
		attribute name : String;
	}

	metadata def RetryPolicy {
		attribute maxAttempts : Integer;
		attribute initialInterval : Duration;
		attribute backoffCoefficient : Real;
		attribute maxInterval : Duration;
	}

	metadata def Timeout {
		attribute startToClose : Duration;
		attribute scheduleToClose : Duration;
		attribute heartbeat : Duration;
	}

	metadata def Deterministic;

	metadata def Idempotent;

	metadata def Schedule {
		attribute spec : String;
		attribute jitter : Duration;
	}

	metadata def ExternalStorage {
		attribute threshold : Integer;
	}
}

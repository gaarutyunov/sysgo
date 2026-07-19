// Order-fulfilment workflow, expressed in SysML v2 textual notation and
// annotated with the sysgo TemporalProfile. `sysgo gen temporal` turns this
// model into the Go code under ./orders (see doc.go's go:generate directive).
package Orders {
	import ScalarValues::*;
	import TemporalProfile::*;

	item def Order {
		attribute id : String;
	}

	// Each @Activity action becomes a method on the generated Activities port,
	// which this example implements by hand in activities.go.
	@Activity { taskQueue = "orders"; }
	action def ChargeCard {
		in order : Order;
	}

	@Activity { taskQueue = "orders"; }
	action def SendReceipt {
		in order : Order;
	}

	// The @Workflow action becomes ProcessOrderWorkflow, orchestrating the
	// activities in declaration order on the "orders" task queue.
	@Workflow { id = "ProcessOrder"; taskQueue = "orders"; }
	action def ProcessOrder {
		in order : Order;
		action charge : ChargeCard;
		action notify : SendReceipt;
	}
}

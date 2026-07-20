// The Order bounded context as a single SysML v2 source that combines all three
// sysgo generators:
//
//   - DDD core   — the domain, ports and PlaceOrder use case (via `sysgo generate`
//                  from model.json; see sysgo.yaml). This is the business logic.
//   - OpenAPI    — a REST entrypoint (@REST) that drives the PlaceOrder use case.
//   - Temporal   — a workflow entrypoint (@Workflow/@Activity) that drives it too.
//
// The OpenAPI and Temporal entrypoints are just transports: they natively import
// and call the DDD business logic. Regenerate with `go generate ./...`.
package OrderContext {
	private import ScalarValues::*;
	import RESTProfile::*;
	import TemporalProfile::*;

	attribute def Money {
		attribute amount : Real;
		attribute currency : String;
	}

	item def ChargeRequest {
		attribute amount : Real;
	}

	item def Receipt {
		attribute id : String;
	}

	port def PaymentPort {
		out item charge : ChargeRequest;
		in item receipt : Receipt;
	}

	port def OrderRepository {
		in item order : Order;
	}

	part def Order {
		attribute id : String;
		attribute total : Money;
		part lines : LineItem[*];
		port pay : PaymentPort;
	}

	part def LineItem {
		attribute sku : String;
		attribute qty : Integer;
	}

	// The DDD use case — the business logic the entrypoints below drive.
	action def PlaceOrder {
		in order : Order;
	}

	// OpenAPI entrypoint: POST /orders, request/response bodies typed by Order.
	@REST { path = "/orders"; method = "POST"; successStatus = 201; }
	action def PlaceOrderAPI {
		in order : Order;
		out placed : Order;
	}

	// Temporal entrypoint: a workflow orchestrating two activities.
	@Activity { taskQueue = "orders"; }
	action def ChargeCard {
		in order : Order;
	}

	@Activity { taskQueue = "orders"; }
	action def SendReceipt {
		in order : Order;
	}

	@Workflow { id = "ProcessOrder"; taskQueue = "orders"; }
	action def ProcessOrder {
		in order : Order;
		action charge : ChargeCard;
		action notify : SendReceipt;
	}
}

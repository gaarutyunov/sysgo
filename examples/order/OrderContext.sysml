// Real SysML v2 textual notation for the Order bounded context.
//
// sysgo consumes the SysML v2 API JSON; transform this source to that JSON with
// real SysML tooling first, e.g.:
//
//   scripts/sysml2json.sh examples/order/OrderContext.sysml examples/order/model.json
//
// then run: sysgo generate -c examples/order/sysgo.yaml
package OrderContext {
	private import ScalarValues::*;

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

	action def PlaceOrder {
		in order : Order;
	}
}

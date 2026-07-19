// A small product-catalog REST API in SysML v2, annotated with the sysgo
// RESTProfile. `sysgo gen openapi` turns this into openapi.yaml (see doc.go),
// and oapi-codegen turns that spec into the gin server + types under ./api.
package Catalog {
	import ScalarValues::*;
	import RESTProfile::*;

	@Api { basePath = "/v1"; version = "1.0.0"; }
	package API {
		item def Product {
			attribute id : String;
			attribute name : String;
			attribute price : Real;
		}

		// GET /products — list the catalog.
		@REST { path = "/products"; method = "GET"; successStatus = 200; }
		action def ListProducts {
			out products : Product;
		}

		// POST /products — create a product from the request body.
		@REST { path = "/products"; method = "POST"; successStatus = 201; }
		action def CreateProduct {
			in product : Product;
			out created : Product;
		}
	}
}

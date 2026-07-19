// A small product-catalog REST API in SysML v2, annotated with the sysgo
// RESTProfile. `sysgo gen openapi` turns this into openapi.yaml (see doc.go),
// and oapi-codegen turns that spec into the gin server + types under ./api.
//
// Operations return a single Product each: the generator maps a directed
// parameter to that type's object schema (it does not model array multiplicity),
// so single-object operations keep the generated schema and the handlers in
// agreement — which the integration test enforces with openapi3filter.
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

		// POST /products — create a product from the request body, echo it back.
		@REST { path = "/products"; method = "POST"; successStatus = 201; }
		action def CreateProduct {
			in product : Product;
			out created : Product;
		}

		// GET /products/featured — return the featured (most recently created) product.
		@REST { path = "/products/featured"; method = "GET"; successStatus = 200; }
		action def GetFeaturedProduct {
			out product : Product;
		}
	}
}

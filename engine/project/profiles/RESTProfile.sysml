// REST metadata profile (specs/OPENAPI.md §3). Bundled with the engine and
// loaded into every workspace so `@REST` / `@Api` / `@ErrorModel` annotations
// resolve in-model. Kept parser-compatible.
package RESTProfile {
	import ScalarValues::*;

	datatype HttpMethod :> String;

	metadata def Api {
		attribute basePath : String;
		attribute version : String;
	}

	metadata def REST {
		attribute path : String;
		attribute method : HttpMethod;
		attribute successStatus : Integer;
		attribute requestContentType : String;
		attribute responseContentType : String;
	}

	metadata def ErrorModel {
		attribute schemaRef : String;
	}
}

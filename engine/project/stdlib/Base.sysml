// Curated, parser-compatible subset of the SysML v2 standard library `Base`
// package. Derived from the OMG SysML v2 release (MIT); see attribution in the
// package doc. This is a working minimal core — the full library bundle is a
// drop-in expansion (the loader reads every embedded .sysml file).
package Base {
	abstract classifier Anything;

	abstract classifier DataValue :> Anything;

	abstract classifier Occurrence :> Anything;
}

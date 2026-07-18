// Curated, parser-compatible subset of the SysML v2 standard library
// `ScalarValues` package. Derived from the OMG SysML v2 release (MIT). The
// numeric tower is modelled with specialization so inheritance-chain resolution
// works (Natural :> Integer :> Real :> Number :> NumericalValue :> ScalarValue).
package ScalarValues {
	abstract datatype ScalarValue :> Base::DataValue;

	datatype Boolean :> ScalarValue;

	datatype String :> ScalarValue;

	abstract datatype NumericalValue :> ScalarValue;

	abstract datatype Number :> NumericalValue;

	datatype Real :> Number;

	datatype Integer :> Real;

	datatype Natural :> Integer;
}

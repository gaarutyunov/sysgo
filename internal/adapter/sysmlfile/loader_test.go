package sysmlfile

import "testing"

func TestDecodeArray(t *testing.T) {
	els, err := Decode([]byte(`[{"@id":"1","@type":"Package"}]`))
	if err != nil || len(els) != 1 || els[0]["@id"] != "1" {
		t.Fatalf("array decode failed: %v %v", els, err)
	}
}

func TestDecodeWrapped(t *testing.T) {
	els, err := Decode([]byte(`{"elements":[{"@id":"2","@type":"Package"}]}`))
	if err != nil || len(els) != 1 || els[0]["@id"] != "2" {
		t.Fatalf("wrapped decode failed: %v %v", els, err)
	}
}

func TestDecodeEmpty(t *testing.T) {
	if _, err := Decode([]byte(`{}`)); err == nil {
		t.Fatal("expected error for empty document")
	}
}

// TestDecodePayloadEnvelope covers the SysML v2 API / pilot SysML2JSON format,
// where each element is wrapped in {"payload": element, "identity": {...}}.
func TestDecodePayloadEnvelope(t *testing.T) {
	els, err := Decode([]byte(`[
		{"payload":{"@id":"a","@type":"PartDefinition","declaredName":"Order"},"identity":{"@id":"a"}},
		{"payload":{"@id":"b","@type":"AttributeUsage","declaredName":"id"},"identity":{"@id":"b"}}
	]`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(els) != 2 || els[0]["@id"] != "a" || els[0]["@type"] != "PartDefinition" {
		t.Fatalf("payload not unwrapped: %+v", els)
	}
	if _, leaked := els[0]["payload"]; leaked {
		t.Fatal("payload wrapper leaked into element")
	}
}

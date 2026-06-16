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

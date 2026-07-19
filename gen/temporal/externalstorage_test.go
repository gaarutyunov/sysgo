package temporal

import (
	"strings"
	"testing"

	"github.com/gaarutyunov/sysgo/engine"
)

const externalStorageModel = `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	@ExternalStorage { threshold = 2048; }
	item def BigPayload { attribute blob : String; }
}`

func TestExternalStorageShape(t *testing.T) {
	m := engine.New().AddFile("app.sysml", externalStorageModel).Build()
	if d := m.Diagnostics(); len(d) != 0 {
		t.Fatalf("model diagnostics: %v", d)
	}
	src, err := GenerateExternalStorage(m, "gen")
	if err != nil {
		t.Fatalf("GenerateExternalStorage: %v", err)
	}
	n := norm(src)
	for _, want := range []string{
		"type ExternalStore interface",
		"Put(ref string, data []byte) error",
		"Get(ref string) ([]byte, error)",
		"type ExternalStorageCodec struct",
		"var _ converter.PayloadCodec = (*ExternalStorageCodec)(nil)",
		"func (c *ExternalStorageCodec) Encode(payloads []*v1.Payload) ([]*v1.Payload, error)",
		"if len(p.Data) <= c.Threshold",
		"func (c *ExternalStorageCodec) Decode(payloads []*v1.Payload) ([]*v1.Payload, error)",
		"func NewDataConverter(store ExternalStore) converter.DataConverter",
		"converter.NewCodecDataConverter(converter.GetDefaultDataConverter()",
		"Threshold: 2048",
	} {
		if !strings.Contains(n, want) {
			t.Errorf("generated external storage missing %q\n---\n%s", want, src)
		}
	}
}

// TestExternalStorageCompiles builds the generated codec + data-converter
// wiring against the Temporal SDK, proving it satisfies converter.PayloadCodec.
func TestExternalStorageCompiles(t *testing.T) {
	m := engine.New().AddFile("app.sysml", externalStorageModel).Build()
	src, err := GenerateExternalStorage(m, "gen")
	if err != nil {
		t.Fatalf("GenerateExternalStorage: %v", err)
	}
	compileFiles(t, map[string]string{"externalstorage.go": src})
}

// TestExternalStorageMinThreshold uses the tightest threshold when several
// payloads declare @ExternalStorage.
func TestExternalStorageMinThreshold(t *testing.T) {
	src := `package App {
	import ScalarValues::*;
	import TemporalProfile::*;
	@ExternalStorage { threshold = 4096; }
	item def A { attribute x : String; }
	@ExternalStorage { threshold = 1024; }
	item def B { attribute y : String; }
}`
	m := engine.New().AddFile("app.sysml", src).Build()
	out, err := GenerateExternalStorage(m, "gen")
	if err != nil {
		t.Fatalf("GenerateExternalStorage: %v", err)
	}
	if !strings.Contains(norm(out), "Threshold: 1024") {
		t.Errorf("expected the tightest threshold (1024):\n%s", out)
	}
}

// TestNoExternalStorage emits only the file header when no @ExternalStorage is
// present.
func TestNoExternalStorage(t *testing.T) {
	m := engine.New().AddFile("m.sysml", "package M { part def X; }").Build()
	out, err := GenerateExternalStorage(m, "gen")
	if err != nil {
		t.Fatalf("GenerateExternalStorage: %v", err)
	}
	if strings.Contains(out, "ExternalStorageCodec") || strings.Contains(out, "NewDataConverter") {
		t.Errorf("unexpected codec for a model without @ExternalStorage:\n%s", out)
	}
}

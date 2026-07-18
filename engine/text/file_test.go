package text

import (
	"fmt"
	"sync"
	"testing"
)

func TestFileSetAddStableAndDistinct(t *testing.T) {
	fs := NewFileSet()

	a1 := fs.Add("a.sysml")
	b := fs.Add("b.sysml")
	a2 := fs.Add("a.sysml")

	if a1 == NoFile || b == NoFile {
		t.Fatalf("Add returned NoFile: a=%d b=%d", a1, b)
	}
	if a1 != a2 {
		t.Errorf("Add(%q) returned %d then %d; want stable", "a.sysml", a1, a2)
	}
	if a1 == b {
		t.Errorf("distinct paths shared FileId %d", a1)
	}
	if got := fs.Path(a1); got != "a.sysml" {
		t.Errorf("Path(%d) = %q, want %q", a1, got, "a.sysml")
	}
	if got := fs.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}
}

func TestFileSetNoFile(t *testing.T) {
	fs := NewFileSet()
	if got := fs.Path(NoFile); got != "" {
		t.Errorf("Path(NoFile) = %q, want %q", got, "")
	}
	if got := fs.Len(); got != 0 {
		t.Errorf("empty FileSet Len() = %d, want 0", got)
	}
}

func TestFileSetLookup(t *testing.T) {
	fs := NewFileSet()
	id := fs.Add("x.sysml")

	if got, ok := fs.Lookup("x.sysml"); !ok || got != id {
		t.Errorf("Lookup(%q) = (%d, %v), want (%d, true)", "x.sysml", got, ok, id)
	}
	if got, ok := fs.Lookup("missing.sysml"); ok || got != NoFile {
		t.Errorf("Lookup(missing) = (%d, %v), want (NoFile, false)", got, ok)
	}
	// Lookup must not register the path.
	if got := fs.Len(); got != 1 {
		t.Errorf("Len() = %d after Lookup of missing path, want 1", got)
	}
}

func TestFileSetPathOutOfRangePanics(t *testing.T) {
	fs := NewFileSet()
	fs.Add("only.sysml")
	assertPanics(t, "Path out of range", func() {
		fs.Path(FileId(999))
	})
}

func TestFileSetConcurrent(t *testing.T) {
	fs := NewFileSet()
	const goroutines = 16
	const files = 100

	var wg sync.WaitGroup
	got := make([]map[string]FileId, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			m := make(map[string]FileId, files)
			for i := 0; i < files; i++ {
				p := fmt.Sprintf("file-%d.sysml", i)
				m[p] = fs.Add(p)
			}
			got[g] = m
		}(g)
	}
	wg.Wait()

	base := got[0]
	for g := 1; g < goroutines; g++ {
		for p, id := range got[g] {
			if base[p] != id {
				t.Fatalf("goroutine %d disagrees on %q: %d vs %d", g, p, id, base[p])
			}
		}
	}
	if got := fs.Len(); got != files {
		t.Fatalf("Len() = %d, want %d", got, files)
	}
}

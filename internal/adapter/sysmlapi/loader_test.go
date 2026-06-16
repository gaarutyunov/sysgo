package sysmlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseNextLink(t *testing.T) {
	h := `<http://x/elements?page=2>; rel="next", <http://x/elements?page=1>; rel="prev"`
	if got := parseNextLink(h); got != "http://x/elements?page=2" {
		t.Fatalf("parseNextLink = %q", got)
	}
	if got := parseNextLink(""); got != "" {
		t.Fatalf("empty link = %q", got)
	}
}

func TestLoadPaged(t *testing.T) {
	var page2 string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"@id": "b", "@type": "PartDefinition"}})
			return
		}
		w.Header().Set("Link", "<"+page2+">; rel=\"next\"")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"@id": "a", "@type": "PartDefinition"}})
	}))
	defer srv.Close()
	page2 = srv.URL + "/projects/p/commits/c/elements?page=2"

	l := New(srv.URL, "p", "c")
	l.PageSize = 1
	els, err := l.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(els) != 2 || els[0]["@id"] != "a" || els[1]["@id"] != "b" {
		t.Fatalf("paging failed: %+v", els)
	}
}

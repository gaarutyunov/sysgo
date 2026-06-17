// Package sysmlapi implements port.ModelLoader over the SysML v2 REST/HTTP API
// Services. It pages through the commit's elements endpoint.
package sysmlapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Loader fetches a model from a SysML v2 API Services endpoint.
type Loader struct {
	BaseURL  string
	Project  string
	Commit   string
	PageSize int
	Client   *http.Client
}

// New constructs a Loader. commit may be empty (⇒ latest on default branch).
func New(baseURL, project, commit string) *Loader {
	return &Loader{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Project:  project,
		Commit:   commit,
		PageSize: 100,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Load implements port.ModelLoader.
func (l *Loader) Load(ctx context.Context) ([]map[string]any, error) {
	commit := l.Commit
	if commit == "" {
		c, err := l.latestCommit(ctx)
		if err != nil {
			return nil, err
		}
		commit = c
	}
	return l.fetchElements(ctx, commit)
}

// latestCommit resolves the most recent commit on the project's default branch.
func (l *Loader) latestCommit(ctx context.Context) (string, error) {
	var commits []map[string]any
	if err := l.getJSON(ctx, fmt.Sprintf("/projects/%s/commits", url.PathEscape(l.Project)), &commits); err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", fmt.Errorf("sysmlapi: project %s has no commits", l.Project)
	}
	// API returns commits newest-first by convention; take the first with an id.
	for _, c := range commits {
		if id, _ := c["@id"].(string); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("sysmlapi: could not resolve latest commit")
}

// fetchElements pages through the commit's elements endpoint following the
// RFC 5988 "next" Link header when present, else page-number fallback.
func (l *Loader) fetchElements(ctx context.Context, commit string) ([]map[string]any, error) {
	base := fmt.Sprintf("/projects/%s/commits/%s/elements",
		url.PathEscape(l.Project), url.PathEscape(commit))

	var all []map[string]any
	next := base + "?page[size]=" + strconv.Itoa(l.PageSize)
	visited := map[string]bool{}

	for next != "" {
		if visited[next] {
			break
		}
		visited[next] = true

		page, link, err := l.getPage(ctx, next)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		next = parseNextLink(link)
		if next == "" && len(page) == l.PageSize {
			// Defensive stop: the server returned a full page but no rel="next"
			// Link header, so we cannot navigate further. Halt rather than risk
			// looping or guessing page parameters for a non-conforming server.
			break
		}
	}
	return all, nil
}

func (l *Loader) getPage(ctx context.Context, path string) ([]map[string]any, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url(path), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := l.Client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("sysmlapi: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("sysmlapi: GET %s: status %d: %s", path, resp.StatusCode, body)
	}
	var page []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, "", fmt.Errorf("sysmlapi: decode %s: %w", path, err)
	}
	return page, resp.Header.Get("Link"), nil
}

func (l *Loader) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url(path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := l.Client.Do(req)
	if err != nil {
		return fmt.Errorf("sysmlapi: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("sysmlapi: GET %s: status %d: %s", path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (l *Loader) url(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return l.BaseURL + path
}

// parseNextLink extracts the URL of a rel="next" Link header value.
func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		segs := strings.Split(strings.TrimSpace(part), ";")
		if len(segs) < 2 {
			continue
		}
		rawURL := strings.Trim(strings.TrimSpace(segs[0]), "<>")
		for _, p := range segs[1:] {
			p = strings.TrimSpace(p)
			if p == `rel="next"` || p == "rel=next" {
				return rawURL
			}
		}
	}
	return ""
}

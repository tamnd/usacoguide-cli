package usacoguide_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/usacoguide-cli/usacoguide"
)

// fakeDirListing is a JSON array mimicking the GitHub API directory listing.
func fakeDirListing(names ...string) []byte {
	var files []map[string]string
	for _, name := range names {
		files = append(files, map[string]string{
			"name": name,
			"path": "content/2_Bronze/" + name,
			"type": "file",
		})
	}
	b, _ := json.Marshal(files)
	return b
}

const fakeMDX = `---
id: time-complexity
title: Time Complexity
author: Benjamin Qi
---

# Time Complexity

Covers big-O notation and its implications for USACO problems.
`

const fakeMDX2 = `---
id: rectangle-geometry
title: Rectangle Geometry
author: Darren Yao
---

# Rectangle Geometry

A common pattern in bronze problems.
`

func newTestClient(srv *httptest.Server) *usacoguide.Client {
	cfg := usacoguide.DefaultConfig()
	// Point both API and raw URLs to the test server.
	cfg.GitHubAPIURL = srv.URL + "/api"
	cfg.RawGitHubURL = srv.URL + "/raw"
	cfg.Rate = 0
	return usacoguide.NewClient(cfg)
}

func TestModules_bronze(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fakeDirListing("time-complexity.mdx", "rectangle-geometry.mdx"))
			return
		}
		// Raw MDX files.
		if strings.HasSuffix(r.URL.Path, "time-complexity.mdx") {
			_, _ = w.Write([]byte(fakeMDX))
		} else {
			_, _ = w.Write([]byte(fakeMDX2))
		}
	}))
	defer srv.Close()

	c := newTestClient(srv)
	modules, err := c.Modules(context.Background(), "bronze", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("got %d modules, want 2", len(modules))
	}
	if modules[0].ID != "time-complexity" {
		t.Errorf("modules[0].ID = %q, want time-complexity", modules[0].ID)
	}
	if modules[0].Title != "Time Complexity" {
		t.Errorf("modules[0].Title = %q, want Time Complexity", modules[0].Title)
	}
	if modules[0].Division != "bronze" {
		t.Errorf("modules[0].Division = %q, want bronze", modules[0].Division)
	}
	if modules[0].Author != "Benjamin Qi" {
		t.Errorf("modules[0].Author = %q, want Benjamin Qi", modules[0].Author)
	}
	if !strings.Contains(modules[0].URL, "bronze") {
		t.Errorf("URL = %q, should contain 'bronze'", modules[0].URL)
	}
}

func TestModules_unknownDivision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Modules(context.Background(), "foo", 0)
	if err == nil {
		t.Fatal("expected error for unknown division")
	}
}

func TestModules_withLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fakeDirListing("time-complexity.mdx", "rectangle-geometry.mdx"))
			return
		}
		_, _ = w.Write([]byte(fakeMDX))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	modules, err := c.Modules(context.Background(), "bronze", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 1 {
		t.Errorf("got %d modules, want 1 (limit)", len(modules))
	}
}

func TestGetModule_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fakeDirListing("time-complexity.mdx"))
			return
		}
		_, _ = w.Write([]byte(fakeMDX))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	m, err := c.GetModule(context.Background(), "time-complexity")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "time-complexity" {
		t.Errorf("ID = %q, want time-complexity", m.ID)
	}
	if m.Title != "Time Complexity" {
		t.Errorf("Title = %q, want Time Complexity", m.Title)
	}
}

func TestGetModule_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetModule(context.Background(), "nonexistent-module")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
}

func TestInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fakeDirListing("time-complexity.mdx", "rectangle-geometry.mdx"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// 6 divisions x 2 files = 12, but we only count mdx files.
	if info.TotalModules == 0 {
		t.Error("TotalModules should not be 0")
	}
	if len(info.Divisions) != 6 {
		t.Errorf("Divisions = %v, want 6 entries", info.Divisions)
	}
	if info.GuideURL == "" {
		t.Error("GuideURL should not be empty")
	}
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fakeDirListing("time-complexity.mdx"))
			return
		}
		_, _ = w.Write([]byte(fakeMDX))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Modules(context.Background(), "bronze", 1)
	if err != nil {
		t.Fatal(err)
	}
}

package usacoguide

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "usacoguide" {
		t.Errorf("Scheme = %q, want usacoguide", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "usacoguide" {
		t.Errorf("Identity.Binary = %q, want usacoguide", info.Identity.Binary)
	}
}

func TestClassify_id(t *testing.T) {
	typ, id, err := Domain{}.Classify("time-complexity")
	if err != nil {
		t.Fatal(err)
	}
	if typ != "module" {
		t.Errorf("typ = %q, want module", typ)
	}
	if id != "time-complexity" {
		t.Errorf("id = %q, want time-complexity", id)
	}
}

func TestClassify_url(t *testing.T) {
	typ, id, err := Domain{}.Classify("https://usaco.guide/bronze/time-complexity")
	if err != nil {
		t.Fatal(err)
	}
	if typ != "module" {
		t.Errorf("typ = %q, want module", typ)
	}
	if id != "time-complexity" {
		t.Errorf("id = %q, want time-complexity", id)
	}
}

func TestLocate_module(t *testing.T) {
	got, err := Domain{}.Locate("module", "time-complexity")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://usaco.guide/time-complexity"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocate_unknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := []byte(`---
id: time-complexity
title: Time Complexity
author: Benjamin Qi
redirects:
  - /bronze/time-complexity
---

# Time Complexity

This module covers big-O notation.`)

	fm := ParseFrontmatter(content)
	if fm["title"] != "Time Complexity" {
		t.Errorf("title = %q, want Time Complexity", fm["title"])
	}
	if fm["author"] != "Benjamin Qi" {
		t.Errorf("author = %q, want Benjamin Qi", fm["author"])
	}
	if fm["id"] != "time-complexity" {
		t.Errorf("id = %q, want time-complexity", fm["id"])
	}
}

func TestParseFrontmatter_empty(t *testing.T) {
	fm := ParseFrontmatter([]byte("# No frontmatter here"))
	if len(fm) != 0 {
		t.Errorf("expected empty map, got %v", fm)
	}
}

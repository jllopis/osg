package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseWordpress(t *testing.T) {
	wxr := `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:excerpt="http://wordpress.org/export/1.2/excerpt/"
     xmlns:content="http://purl.org/rss/1.0/modules/content/"
     xmlns:wp="http://wordpress.org/export/">
<channel>
  <item>
    <title>Hello World</title>
    <wp:post_date>2024-06-15 10:30:00</wp:post_date>
    <wp:post_name>hello-world</wp:post_name>
    <wp:post_type>post</wp:post_type>
    <wp:status>publish</wp:status>
    <content:encoded><![CDATA[<p>This is <strong>bold</strong> and <a href="https://example.com">a link</a>.</p>]]></content:encoded>
    <category domain="post_tag"><![CDATA[golang]]></category>
    <category domain="post_tag"><![CDATA[web]]></category>
    <category domain="category"><![CDATA[Tech]]></category>
  </item>
  <item>
    <title>Draft Post</title>
    <wp:post_date>2024-07-01 08:00:00</wp:post_date>
    <wp:post_name>draft-post</wp:post_name>
    <wp:post_type>post</wp:post_type>
    <wp:status>draft</wp:status>
    <content:encoded><![CDATA[<p>Draft content</p>]]></content:encoded>
  </item>
  <item>
    <title>A Page</title>
    <wp:post_type>page</wp:post_type>
    <wp:status>publish</wp:status>
    <content:encoded><![CDATA[<p>Page content</p>]]></content:encoded>
  </item>
</channel>
</rss>`

	dir := t.TempDir()
	path := filepath.Join(dir, "export.xml")
	if err := os.WriteFile(path, []byte(wxr), 0o644); err != nil {
		t.Fatal(err)
	}

	posts, err := ParseWordpress(path)
	if err != nil {
		t.Fatalf("ParseWordpress: %v", err)
	}

	// Should skip the "page" type item.
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}

	p := posts[0]
	if p.Title != "Hello World" {
		t.Errorf("title = %q", p.Title)
	}
	if p.Slug != "hello-world" {
		t.Errorf("slug = %q", p.Slug)
	}
	if p.Draft {
		t.Error("expected published post")
	}
	if len(p.Tags) != 2 {
		t.Errorf("tags = %v", p.Tags)
	}
	if len(p.Categories) != 1 || p.Categories[0] != "Tech" {
		t.Errorf("categories = %v", p.Categories)
	}
	if !strings.Contains(p.Content, "**bold**") {
		t.Errorf("content missing bold conversion: %s", p.Content)
	}
	if !strings.Contains(p.Content, "[a link](https://example.com)") {
		t.Errorf("content missing link conversion: %s", p.Content)
	}

	d := posts[1]
	if !d.Draft {
		t.Error("expected draft post")
	}
}

func TestParseHugo_YAML(t *testing.T) {
	dir := t.TempDir()
	postDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `---
title: "Hugo Post"
date: 2024-03-15
tags:
  - go
  - hugo
categories:
  - tech
draft: false
---

This is a Hugo post.
`
	if err := os.WriteFile(filepath.Join(postDir, "my-post.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	posts, err := ParseHugo(dir)
	if err != nil {
		t.Fatalf("ParseHugo: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}

	p := posts[0]
	if p.Title != "Hugo Post" {
		t.Errorf("title = %q", p.Title)
	}
	if p.Date.Year() != 2024 || p.Date.Month() != 3 || p.Date.Day() != 15 {
		t.Errorf("date = %v", p.Date)
	}
	if len(p.Tags) != 2 {
		t.Errorf("tags = %v", p.Tags)
	}
	if p.Draft {
		t.Error("expected published post")
	}
	if !strings.Contains(p.Content, "This is a Hugo post.") {
		t.Errorf("content = %q", p.Content)
	}
}

func TestParseHugo_TOML(t *testing.T) {
	dir := t.TempDir()
	content := `+++
title = "TOML Post"
date = "2024-01-10"
tags = ["rust", "wasm"]
draft = true
+++

TOML frontmatter content.
`
	if err := os.WriteFile(filepath.Join(dir, "toml-post.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	posts, err := ParseHugo(dir)
	if err != nil {
		t.Fatalf("ParseHugo: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}

	p := posts[0]
	if p.Title != "TOML Post" {
		t.Errorf("title = %q", p.Title)
	}
	if !p.Draft {
		t.Error("expected draft post")
	}
	if len(p.Tags) != 2 {
		t.Errorf("tags = %v, want [rust wasm]", p.Tags)
	}
}

func TestPostOutputPath(t *testing.T) {
	p := Post{
		Title: "My Great Post",
		Slug:  "my-great-post",
		Date:  mustParseDate("2024-06-15"),
	}
	if got := p.OutputPath(); got != "2024/06/my-great-post.md" {
		t.Errorf("OutputPath = %q", got)
	}
}

func TestPostToMarkdown(t *testing.T) {
	p := Post{
		Title:      "Test",
		Date:       mustParseDate("2024-01-01"),
		Tags:       []string{"go"},
		Categories: []string{"tech"},
		Content:    "Hello world",
		Draft:      true,
	}
	md := p.ToMarkdown()
	if !strings.Contains(md, "title: \"Test\"") {
		t.Errorf("missing title in:\n%s", md)
	}
	if !strings.Contains(md, "publish: draft") {
		t.Errorf("missing draft in:\n%s", md)
	}
	if !strings.Contains(md, "Hello world") {
		t.Errorf("missing content in:\n%s", md)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "hello-world"},
		{"  Leading spaces  ", "leading-spaces"},
		{"Special!@#Characters", "specialcharacters"},
		{"Multiple   Spaces", "multiple-spaces"},
	}
	for _, tt := range tests {
		if got := slugify(tt.input); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func mustParseDate(s string) (t time.Time) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

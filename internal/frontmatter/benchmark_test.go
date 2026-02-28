package frontmatter

import "testing"

var sampleDoc = []byte(`---
title: "A Sample Article About Performance"
created: 2025-01-15
tags:
  - golang
  - performance
  - benchmarks
osg:
  publish: "published"
  featured: true
  image: "hero.jpg"
  menu: false
  abstract: "This article explores performance optimization techniques."
  author: "Test Author"
---

# Introduction

This is the body of the article. It contains **markdown** content
that follows the frontmatter block.

## Performance

Here we discuss various aspects of performance optimization...
`)

func BenchmarkSplitFrontmatter(b *testing.B) {
	for b.Loop() {
		_, _, _, _ = SplitFrontmatter(sampleDoc)
	}
}

func BenchmarkSplitFrontmatter_NoFrontmatter(b *testing.B) {
	body := []byte("# Just a heading\n\nSome content without frontmatter.")
	for b.Loop() {
		_, _, _, _ = SplitFrontmatter(body)
	}
}

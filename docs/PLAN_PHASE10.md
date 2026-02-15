# Plan: Phase 10 — Polish, Feeds, Doctor & Image Optimization

> Consolidation phase: improve quality, add missing features, optimize assets.

## Context

Phases 1-9 are complete. The summary feature (auto-extract + Kairos placeholder)
and featured post overlay design were added between Phase 9 and Phase 10.

## Steps

### Step 1: Documentation sync (~15 min)
- Update `ROADMAP.md`: mark Phase 9 as done, add summary feature, add Phase 10
- Update `TASKS.md`: mark Phase 9 + summary as done, add new tasks
- Update `DESIGN.md`: add summary pipeline, `summary_strategy` config, featured overlay

### Step 2: Global site RSS/Atom feed (~30 min)
- Generate site-wide `atom.xml` and/or `rss.xml` at the root
- All pages sorted by date, capped at configurable limit (default 20)
- Reuse existing feed templates + `feedPages()` logic from taxonomy feeds
- Add `<link rel="alternate">` in `head.html` partial
- Config option: `site_feed: true` (default) to enable/disable

### Step 3: Doctor improvements (~45 min)
- Make diagnostics actionable: each warning/error gets a concrete fix suggestion
- Add checks: missing templates, unreferenced images, broken wikilinks, empty sections, large images
- Differentiate severity better between dev and prod profiles
- Structured output: group by category (config, content, assets, templates)

### Step 4: Theme polish (~1-2h, iterative)
- Review typography spacing on all page types (index, section, page, taxonomy, 404)
- Improve card design in section listings
- Better responsive behavior at tablet breakpoint (~768px)
- Fine-tune dark mode contrast
- Consistent hover states and transitions
- Review featured overlay on mobile (small images)

### Step 5: TUI + build tests (~1h)
- `internal/build/`: test fillSummaries, configView, outputHTMLPath, buildURL, helpers
- `internal/tui/`: test command parsing, autocomplete matching, key bindings, message handling
- Target: meaningful coverage for the two untested core packages

### Step 6: Image optimization (~1.5-2h)
- WebP conversion using Go's `image` + exec `cwebp` if available
- Responsive `srcset` variants (640w, 1200w)
- Config: `image_optimization: true`, `image_quality: 80`, `image_formats: [webp, original]`
- Update templates to use `<picture>` + `srcset` when optimized variants exist
- Keep placeholder SVGs as-is (already tiny)

## Deferred items (future phases)

These items were evaluated but deferred for now:

- **Kairos AI summaries**: wire `KairosProvider` when Kairos library is stable enough.
  Placeholder code and detailed implementation guide already in `internal/summary/summary.go`.
- **i18n**: multi-language content and template support. Requires significant
  architecture decisions (URL scheme, content organization, template fallbacks).
- **Image gallery / lightbox**: optional JS component for image-heavy posts.
  Low priority until image optimization is in place.
- **`osg new` command**: quick-create a new post from CLI with scaffolded frontmatter.
  Nice-to-have convenience feature.

## Dependencies

- Steps 1-3 are independent quick wins.
- Step 4 (theme polish) is iterative and benefits from visual review.
- Step 5 (tests) can run alongside other steps.
- Step 6 (image optimization) is self-contained and has the most complexity.

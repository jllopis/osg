package build

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// pluginIndex mirrors the structure of plugins-index.json.
type pluginIndex struct {
	Plugins []pluginEntry `json:"plugins"`
}

type pluginEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Repo        string   `json:"repo"`
	Version     string   `json:"version"`
	Hooks       []string `json:"hooks"`
}

// GenerateMarketplace reads plugins-index.json and writes a static HTML page
// at <publicDir>/plugins/index.html with a searchable, filterable listing.
func GenerateMarketplace(indexPath, publicDir string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read plugin index: %w", err)
	}

	var idx pluginIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("parse plugin index: %w", err)
	}

	// Collect all unique hooks for the filter buttons.
	hookSet := map[string]bool{}
	for _, p := range idx.Plugins {
		for _, h := range p.Hooks {
			hookSet[h] = true
		}
	}
	hooks := make([]string, 0, len(hookSet))
	for h := range hookSet {
		hooks = append(hooks, h)
	}
	sort.Strings(hooks)

	outDir := filepath.Join(publicDir, "plugins")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir plugins: %w", err)
	}

	outPath := filepath.Join(outDir, "index.html")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create marketplace html: %w", err)
	}

	writeErr := func() error {
		tmpl, err := template.New("marketplace").Funcs(template.FuncMap{
			"joinHooks": func(hooks []string) string { return strings.Join(hooks, ", ") },
		}).Parse(marketplaceTmpl)
		if err != nil {
			return fmt.Errorf("parse marketplace template: %w", err)
		}
		return tmpl.Execute(f, map[string]any{
			"Plugins": idx.Plugins,
			"Hooks":   hooks,
		})
	}()
	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	return writeErr
}

const marketplaceTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OSG Plugin Marketplace</title>
<style>
:root {
  --bg: #eceff4; --fg: #2e3440; --card: #fff; --border: #d8dee9;
  --accent: #5e81ac; --accent-light: #81a1c1; --tag-bg: #e5e9f0;
  --code-bg: #3b4252; --code-fg: #eceff4;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #2e3440; --fg: #eceff4; --card: #3b4252; --border: #4c566a;
    --accent: #88c0d0; --accent-light: #81a1c1; --tag-bg: #434c5e;
    --code-bg: #2e3440; --code-fg: #d8dee9;
  }
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background: var(--bg); color: var(--fg); line-height: 1.6; padding: 2rem 1rem; }
.container { max-width: 900px; margin: 0 auto; }
h1 { font-size: 1.8rem; margin-bottom: 0.5rem; }
.subtitle { color: var(--accent-light); margin-bottom: 1.5rem; }
.controls { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 1.5rem; }
.search-input { flex: 1; min-width: 200px; padding: 0.5rem 0.75rem; border: 1px solid var(--border);
  border-radius: 6px; background: var(--card); color: var(--fg); font-size: 0.95rem; }
.filter-btn { padding: 0.35rem 0.75rem; border: 1px solid var(--border); border-radius: 16px;
  background: var(--card); color: var(--fg); cursor: pointer; font-size: 0.85rem; transition: all 0.15s; }
.filter-btn:hover, .filter-btn.active { background: var(--accent); color: #fff; border-color: var(--accent); }
.grid { display: grid; grid-template-columns: 1fr; gap: 1rem; }
@media (min-width: 600px) { .grid { grid-template-columns: 1fr 1fr; } }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem;
  transition: box-shadow 0.15s; }
.card:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
.card-name { font-size: 1.15rem; font-weight: 600; color: var(--accent); }
.card-version { font-size: 0.85rem; color: var(--accent-light); margin-left: 0.5rem; }
.card-desc { margin: 0.5rem 0; font-size: 0.95rem; }
.card-author { font-size: 0.85rem; color: var(--accent-light); }
.card-hooks { display: flex; flex-wrap: wrap; gap: 0.35rem; margin-top: 0.5rem; }
.hook-tag { font-size: 0.75rem; padding: 0.15rem 0.5rem; background: var(--tag-bg);
  border-radius: 10px; white-space: nowrap; }
.card-install { margin-top: 0.75rem; }
.card-install code { display: block; background: var(--code-bg); color: var(--code-fg);
  padding: 0.5rem 0.75rem; border-radius: 4px; font-size: 0.85rem; overflow-x: auto;
  white-space: nowrap; cursor: pointer; position: relative; }
.card-install code:hover::after { content: "click to copy"; position: absolute; right: 0.5rem;
  top: 50%; transform: translateY(-50%); font-size: 0.7rem; color: var(--accent-light); }
.no-results { text-align: center; padding: 2rem; color: var(--accent-light); display: none; }
</style>
</head>
<body>
<div class="container">
  <h1>OSG Plugin Marketplace</h1>
  <p class="subtitle">Extend your site with WASM plugins</p>
  <div class="controls">
    <input type="search" class="search-input" id="search" placeholder="Search plugins..." autocomplete="off">
    <button class="filter-btn active" data-hook="all">All</button>
    {{- range .Hooks}}
    <button class="filter-btn" data-hook="{{.}}">{{.}}</button>
    {{- end}}
  </div>
  <div class="grid" id="grid">
    {{- range .Plugins}}
    <div class="card" data-name="{{.Name}}" data-desc="{{.Description}}" data-author="{{.Author}}" data-hooks="{{joinHooks .Hooks}}">
      <div>
        <span class="card-name">{{.Name}}</span>
        <span class="card-version">v{{.Version}}</span>
      </div>
      <p class="card-desc">{{.Description}}</p>
      <div class="card-author">by {{.Author}}</div>
      <div class="card-hooks">
        {{- range .Hooks}}
        <span class="hook-tag">{{.}}</span>
        {{- end}}
      </div>
      <div class="card-install">
        <code onclick="navigator.clipboard.writeText(this.textContent.trim())">osg plugin install {{.Repo}}</code>
      </div>
    </div>
    {{- end}}
  </div>
  <p class="no-results" id="no-results">No plugins match your search.</p>
</div>
<script>
(function(){
  const search = document.getElementById('search');
  const grid = document.getElementById('grid');
  const cards = Array.from(grid.querySelectorAll('.card'));
  const btns = document.querySelectorAll('.filter-btn');
  const noResults = document.getElementById('no-results');
  let activeHook = 'all';

  function filter() {
    const q = search.value.toLowerCase().trim();
    let visible = 0;
    cards.forEach(c => {
      const text = (c.dataset.name + ' ' + c.dataset.desc + ' ' + c.dataset.author).toLowerCase();
      const hooks = c.dataset.hooks;
      const matchSearch = !q || text.includes(q);
      const matchHook = activeHook === 'all' || hooks.includes(activeHook);
      const show = matchSearch && matchHook;
      c.style.display = show ? '' : 'none';
      if (show) visible++;
    });
    noResults.style.display = visible === 0 ? '' : 'none';
  }

  search.addEventListener('input', filter);
  btns.forEach(btn => {
    btn.addEventListener('click', () => {
      btns.forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      activeHook = btn.dataset.hook;
      filter();
    });
  });
})();
</script>
</body>
</html>
`

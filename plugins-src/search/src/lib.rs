use serde::Deserialize;
use serde_json::{json, Value};
use std::fs;
use std::mem;
use std::path::Path;

#[no_mangle]
pub extern "C" fn alloc(len: i32) -> i32 {
    let mut buf = Vec::<u8>::with_capacity(len as usize);
    let ptr = buf.as_mut_ptr();
    mem::forget(buf);
    ptr as i32
}

#[no_mangle]
pub unsafe extern "C" fn dealloc(ptr: i32, len: i32) {
    let _ = Vec::from_raw_parts(ptr as *mut u8, len as usize, len as usize);
}

#[derive(Deserialize)]
struct Event {
    #[serde(rename = "type")]
    event_type: String,
    payload: Value,
}

#[no_mangle]
pub unsafe extern "C" fn handle_event(ptr: i32, len: i32) -> u64 {
    if ptr == 0 || len == 0 {
        return 0;
    }

    let slice = std::slice::from_raw_parts(ptr as *const u8, len as usize);
    let input = String::from_utf8_lossy(slice);
    let event: Event = match serde_json::from_str(&input) {
        Ok(event) => event,
        Err(_) => return 0,
    };

    if event.event_type != "build.finished" {
        return 0;
    }

    if let Err(e) = write_search(&event.payload) {
        eprintln!("osg-search plugin: {}", e);
    }
    0
}

/// Strip HTML tags and normalize whitespace for plain text indexing.
/// This reduces JSON size by ~40-60% compared to full HTML.
fn strip_html(html: &str) -> String {
    let mut result = String::with_capacity(html.len());
    let mut in_tag = false;
    let mut prev_char = ' ';

    for c in html.chars() {
        if c == '<' {
            in_tag = true;
            continue;
        }
        if c == '>' {
            in_tag = false;
            continue;
        }
        if !in_tag {
            // Normalize multiple whitespace to single space
            if c.is_whitespace() {
                if prev_char != ' ' {
                    result.push(' ');
                    prev_char = ' ';
                }
            } else {
                result.push(c);
                prev_char = c;
            }
        }
    }

    result.trim().to_string()
}

/// Extract section name from permalink path.
/// "/blog/2025/01/01/post/" → "blog", "/2025/01/01/post/" → ""
fn extract_section(permalink: &str) -> String {
    let trimmed = permalink.trim_matches('/');
    match trimmed.split('/').next() {
        Some(seg)
            if !seg.is_empty()
                && !(seg.len() == 4 && seg.chars().all(|c| c.is_ascii_digit())) =>
        {
            seg.to_string()
        }
        _ => String::new(),
    }
}

fn write_search(payload: &Value) -> Result<(), String> {
    let config = payload.get("config").ok_or("no config")?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .unwrap_or("public");

    let pages = payload
        .get("site")
        .and_then(|v| v.get("pages"))
        .and_then(|v| v.as_array())
        .ok_or("no pages")?;

    let mut items = Vec::new();
    for page in pages {
        let title = page.get("title").and_then(|v| v.as_str()).unwrap_or("");
        let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");
        let content_html = page.get("content").and_then(|v| v.as_str()).unwrap_or("");
        let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
        let date = page.get("date").and_then(|v| v.as_str()).unwrap_or("");
        let taxonomies = page.get("taxonomies").unwrap_or(&Value::Null);
        let section = extract_section(permalink);

        // Strip HTML for plain text search index
        let content_text = strip_html(content_html);

        items.push(json!({
            "id": permalink,
            "title": title,
            "summary": summary,
            "content": content_text,
            "permalink": permalink,
            "date": date,
            "section": section,
            "taxonomies": taxonomies
        }));
    }

    let search = json!({ "items": items });

    let search_path = Path::new(public_dir).join("search.json");
    fs::write(&search_path, search.to_string())
        .map_err(|e| format!("write {:?}: {}", search_path, e))?;

    let html_dir = Path::new(public_dir).join("search");
    fs::create_dir_all(&html_dir).map_err(|e| format!("mkdir {:?}: {}", html_dir, e))?;
    let html_path = html_dir.join("index.html");
    fs::write(&html_path, search_html()).map_err(|e| format!("write {:?}: {}", html_path, e))?;

    let js_dir = Path::new(public_dir).join("js");
    fs::create_dir_all(&js_dir).map_err(|e| format!("mkdir {:?}: {}", js_dir, e))?;
    let js_path = js_dir.join("search.js");
    fs::write(&js_path, search_js()).map_err(|e| format!("write {:?}: {}", js_path, e))?;

    Ok(())
}

fn search_js() -> String {
    r##"// MiniSearch is loaded from CDN in the HTML
// This module wraps it with OSG-specific functionality

class OSGSearch {
  constructor(options = {}) {
    this.inputSelector = options.inputSelector || '#search-input';
    this.resultsSelector = options.resultsSelector || '#search-results';
    this.maxResults = options.maxResults || 10;
    this.minChars = options.minChars || 2;
    this.onResults = options.onResults || null;
    this.miniSearch = null;
    this.documents = [];
    this.docMap = new Map();
    this.filters = { dateFrom: '', dateTo: '', tags: [], section: '', sortBy: 'relevance' };
    this.ready = this.init();
  }

  async init() {
    this.input = document.querySelector(this.inputSelector);
    this.results = document.querySelector(this.resultsSelector);
    if (!this.input) return;

    await this.loadIndex();
    this.bindEvents();
  }

  async loadIndex() {
    try {
      const response = await fetch('/search.json');
      const data = await response.json();
      this.documents = data.items || [];
      this.docMap = new Map(this.documents.map(d => [d.permalink, d]));

      // Initialize MiniSearch with indexed fields
      this.miniSearch = new MiniSearch({
        fields: ['title', 'summary', 'content', 'tags'],
        storeFields: ['title', 'permalink', 'date', 'summary', 'section'],
        searchOptions: {
          boost: { title: 3, summary: 2, tags: 1.5 },
          fuzzy: 0.2,
          prefix: true
        }
      });

      // Add documents with flattened tags for indexing
      const docsToAdd = this.documents.map(doc => ({
        id: doc.permalink,
        title: doc.title || '',
        summary: doc.summary || '',
        content: doc.content || '',
        tags: this.extractTags(doc.taxonomies),
        permalink: doc.permalink,
        date: doc.date || '',
        section: doc.section || ''
      }));

      this.miniSearch.addAll(docsToAdd);
    } catch (err) {
      console.error('Failed to load search index:', err);
    }
  }

  extractTags(taxonomies) {
    if (!taxonomies || typeof taxonomies !== 'object') return '';
    const allTags = [];
    for (const terms of Object.values(taxonomies)) {
      if (Array.isArray(terms)) {
        allTags.push(...terms);
      }
    }
    return allTags.join(' ');
  }

  extractTagsArray(taxonomies) {
    if (!taxonomies || typeof taxonomies !== 'object') return [];
    const all = [];
    for (const terms of Object.values(taxonomies)) {
      if (Array.isArray(terms)) all.push(...terms.map(t => t.toLowerCase()));
    }
    return all;
  }

  setFilters(filters) {
    Object.assign(this.filters, filters);
    if (this.input && this.input.value.length >= this.minChars) {
      this.search(this.input.value);
    }
  }

  getAllTags() {
    const counts = {};
    for (const doc of this.documents) {
      for (const tag of this.extractTagsArray(doc.taxonomies)) {
        counts[tag] = (counts[tag] || 0) + 1;
      }
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1]).map(e => e[0]);
  }

  getAllSections() {
    const set = new Set();
    for (const doc of this.documents) {
      if (doc.section) set.add(doc.section);
    }
    return [...set].sort();
  }

  bindEvents() {
    let debounceTimer;
    this.input.addEventListener('input', (e) => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => this.search(e.target.value), 150);
    });

    this.input.addEventListener('focus', () => {
      if (this.input.value.length >= this.minChars) {
        this.search(this.input.value);
      }
    });

    document.addEventListener('click', (e) => {
      if (!this.input.contains(e.target) && !this.results.contains(e.target)) {
        this.hide();
      }
    });

    // Keyboard navigation from input
    this.input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        this.hide();
        this.input.blur();
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        const first = this.results.querySelector('a');
        if (first) first.focus();
      }
      if (e.key === 'Enter') {
        const first = this.results.querySelector('a');
        if (first) window.location = first.href;
      }
    });

    // Keyboard navigation within results
    this.results.addEventListener('keydown', (e) => {
      const links = [...this.results.querySelectorAll('a')];
      if (links.length === 0) return;
      const current = document.activeElement;
      const idx = links.indexOf(current);
      if (idx === -1) return;

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (idx < links.length - 1) links[idx + 1].focus();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (idx > 0) links[idx - 1].focus();
        else this.input.focus();
      } else if (e.key === 'Enter') {
        e.preventDefault();
        window.location = current.href;
      } else if (e.key === 'Escape') {
        this.hide();
        this.input.focus();
      }
    });
  }

  search(query) {
    if (!this.miniSearch || query.length < this.minChars) {
      this.hide();
      if (this.onResults) this.onResults(0, 0);
      return;
    }

    try {
      let results = this.miniSearch.search(query, { prefix: true, fuzzy: 0.2 });
      const total = results.length;
      results = this.applyFilters(results);
      results = this.applySorting(results);
      const shown = Math.min(results.length, this.maxResults);
      results = results.slice(0, this.maxResults);
      this.render(results, query);
      if (this.onResults) this.onResults(shown, total);
    } catch (err) {
      this.results.innerHTML = '<div class="search-no-results">No results for "' + this.escapeHtml(query) + '"</div>';
      this.show();
      if (this.onResults) this.onResults(0, 0);
    }
  }

  applyFilters(results) {
    const { dateFrom, dateTo, tags, section } = this.filters;
    if (!dateFrom && !dateTo && tags.length === 0 && !section) return results;

    return results.filter(r => {
      const doc = this.docMap.get(r.id);
      if (!doc) return true;
      if (dateFrom && doc.date < dateFrom) return false;
      if (dateTo && doc.date > dateTo) return false;
      if (section && (doc.section || '') !== section) return false;
      if (tags.length > 0) {
        const docTags = this.extractTagsArray(doc.taxonomies);
        if (!tags.every(t => docTags.includes(t))) return false;
      }
      return true;
    });
  }

  applySorting(results) {
    if (this.filters.sortBy === 'date') {
      return [...results].sort((a, b) => {
        const da = this.docMap.get(a.id);
        const db = this.docMap.get(b.id);
        return (db && db.date || '').localeCompare(da && da.date || '');
      });
    }
    return results; // MiniSearch default: relevance score
  }

  render(results, query) {
    if (results.length === 0) {
      this.results.innerHTML = '<div class="search-no-results">No results for "' + this.escapeHtml(query) + '"</div>';
      this.show();
      return;
    }

    const html = results.map(r => {
      const doc = this.docMap.get(r.id) || {};
      const excerpt = this.getExcerpt(doc, query);
      const section = doc.section ? '<span class="search-result-section">' + this.escapeHtml(doc.section) + '</span>' : '';
      return '<li class="search-result"><a href="' + r.id + '" tabindex="0">'
        + '<span class="search-result-title">' + this.highlight(doc.title || '', query) + '</span>'
        + '<span class="search-result-meta">' + section + '<span class="search-result-date">' + (doc.date || '') + '</span></span>'
        + '<span class="search-result-excerpt">' + excerpt + '</span>'
        + '</a></li>';
    }).join('');

    this.results.innerHTML = '<ul class="search-results-list" role="listbox">' + html + '</ul>';
    this.show();
  }

  getExcerpt(doc, query) {
    const content = doc.content || '';
    const terms = query.toLowerCase().split(/\s+/).filter(t => t);
    const firstTerm = terms[0] || '';

    let excerpt = '';
    const pos = content.toLowerCase().indexOf(firstTerm);
    if (pos !== -1) {
      const start = Math.max(0, pos - 40);
      const end = Math.min(content.length, pos + firstTerm.length + 80);
      excerpt = (start > 0 ? '\u2026' : '') + content.slice(start, end) + (end < content.length ? '\u2026' : '');
    } else if (doc.summary) {
      excerpt = doc.summary.slice(0, 120) + (doc.summary.length > 120 ? '\u2026' : '');
    } else {
      excerpt = content.slice(0, 120) + (content.length > 120 ? '\u2026' : '');
    }

    return this.highlight(excerpt, query);
  }

  highlight(text, query) {
    const terms = query.toLowerCase().split(/\s+/).filter(t => t);
    let result = this.escapeHtml(text);
    for (const term of terms) {
      const regex = new RegExp('(' + this.escapeRegex(term) + ')', 'gi');
      result = result.replace(regex, '<mark>$1</mark>');
    }
    return result;
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  escapeRegex(str) {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  show() {
    this.results.classList.add('visible');
    this.results.setAttribute('aria-expanded', 'true');
  }

  hide() {
    this.results.classList.remove('visible');
    this.results.setAttribute('aria-expanded', 'false');
  }
}

window.OSGSearch = OSGSearch;"##.to_string()
}

fn search_html() -> String {
    r##"<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Search</title>
    <style>
      :root {
        --nord0: #2e3440; --nord1: #3b4252; --nord3: #434c5e; --nord4: #d8dee9; --nord6: #eceff4;
        --nord8: #88c0d0; --nord10: #5e81ac; --nord13: #ebcb8b;
      }
      @media (prefers-color-scheme: light) {
        :root { --bg: var(--nord6); --text: var(--nord0); --muted: #7b88a1; --surface: #dde2ec; --accent: var(--nord10); --accent-text: #fff; }
      }
      @media (prefers-color-scheme: dark) {
        :root { --bg: var(--nord0); --text: var(--nord4); --muted: #6b7894; --surface: var(--nord1); --accent: var(--nord8); --accent-text: var(--nord0); }
      }
      *, *::before, *::after { box-sizing: border-box; }
      body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); margin: 0; padding: 2rem; min-height: 100vh; }
      .search-container { max-width: 800px; margin: 0 auto; }
      h1 { margin-bottom: 1.5rem; }

      /* Search input */
      #search-input { width: 100%; padding: 1rem 1.25rem; font-size: 1.25rem; border: 2px solid var(--surface); border-radius: 8px; background: var(--bg); color: var(--text); outline: none; transition: border-color 0.2s; }
      #search-input:focus { border-color: var(--accent); }
      #search-input::placeholder { color: var(--muted); }

      /* Filters */
      .search-filters { margin: 1rem 0; border: 1px solid var(--surface); border-radius: 8px; }
      .search-filters summary { cursor: pointer; color: var(--accent); font-weight: 600; padding: 0.75rem 1rem; user-select: none; list-style: none; display: flex; align-items: center; gap: 0.5rem; }
      .search-filters summary::-webkit-details-marker { display: none; }
      .search-filters summary::before { content: '\25B6'; font-size: 0.7rem; transition: transform 0.2s; }
      .search-filters[open] summary::before { transform: rotate(90deg); }
      .search-filters-body { padding: 0 1rem 1rem; }
      .filter-row { display: flex; flex-wrap: wrap; gap: 1rem; margin-bottom: 0.75rem; }
      .filter-group { display: flex; flex-direction: column; gap: 0.25rem; }
      .filter-group label { font-size: 0.75rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
      .filter-group input[type="date"],
      .filter-group select { padding: 0.5rem 0.625rem; border: 1px solid var(--surface); border-radius: 6px; background: var(--bg); color: var(--text); font-size: 0.875rem; }
      .filter-group input[type="date"]:focus,
      .filter-group select:focus { border-color: var(--accent); outline: none; }
      .filter-tags { display: flex; flex-wrap: wrap; gap: 0.375rem; max-height: 150px; overflow-y: auto; padding: 0.25rem 0; }
      .filter-tag { padding: 0.2rem 0.6rem; border: 1px solid var(--surface); border-radius: 99px; background: transparent; color: var(--text); font-size: 0.8rem; cursor: pointer; transition: all 0.15s; }
      .filter-tag:hover { border-color: var(--accent); }
      .filter-tag.active { background: var(--accent); color: var(--accent-text); border-color: var(--accent); }

      /* Toolbar: sort + stats */
      .search-toolbar { display: flex; align-items: center; justify-content: space-between; margin: 1rem 0; gap: 1rem; flex-wrap: wrap; }
      .search-sort { display: flex; }
      .sort-btn { padding: 0.375rem 0.75rem; border: 1px solid var(--surface); background: transparent; color: var(--text); font-size: 0.85rem; cursor: pointer; transition: all 0.15s; }
      .sort-btn:first-child { border-radius: 6px 0 0 6px; }
      .sort-btn:last-child { border-radius: 0 6px 6px 0; border-left: none; }
      .sort-btn.active { background: var(--accent); color: var(--accent-text); border-color: var(--accent); }
      .sort-btn:hover:not(.active) { border-color: var(--accent); }
      .search-stats { color: var(--muted); font-size: 0.9rem; }

      /* Results */
      #search-results ul { list-style: none; padding: 0; margin: 0; }
      .search-result { margin: 0.75rem 0; padding: 1rem; border-radius: 8px; background: var(--surface); transition: transform 0.15s; }
      .search-result:hover { transform: translateX(4px); }
      .search-result a { text-decoration: none; color: inherit; display: block; outline: none; }
      .search-result a:focus { outline: 2px solid var(--accent); outline-offset: -2px; border-radius: 6px; }
      .search-result-title { font-size: 1.1rem; font-weight: 600; color: var(--accent); display: block; }
      .search-result-meta { display: flex; align-items: center; gap: 0.5rem; margin: 0.25rem 0; }
      .search-result-section { font-size: 0.7rem; padding: 0.1rem 0.5rem; border-radius: 99px; background: var(--accent); color: var(--accent-text); font-weight: 600; text-transform: uppercase; letter-spacing: 0.03em; }
      .search-result-date { font-size: 0.85rem; color: var(--muted); }
      .search-result-excerpt { font-size: 0.95rem; line-height: 1.5; display: block; margin-top: 0.25rem; }
      mark { background: var(--nord13); color: var(--nord0); padding: 0 0.2em; border-radius: 2px; }
      .search-no-results { padding: 2rem; text-align: center; color: var(--muted); }

      @media (max-width: 600px) {
        body { padding: 1rem; }
        .filter-row { flex-direction: column; }
        .search-toolbar { flex-direction: column; align-items: flex-start; }
      }
    </style>
  </head>
  <body>
    <div class="search-container">
      <h1>Search</h1>
      <input id="search-input" type="search" placeholder="Type to search..." autocomplete="off" autofocus />

      <details class="search-filters" id="search-filters">
        <summary>Filters</summary>
        <div class="search-filters-body">
          <div class="filter-row">
            <div class="filter-group">
              <label for="filter-date-from">From</label>
              <input type="date" id="filter-date-from" />
            </div>
            <div class="filter-group">
              <label for="filter-date-to">To</label>
              <input type="date" id="filter-date-to" />
            </div>
            <div class="filter-group" id="filter-section-group">
              <label for="filter-section">Section</label>
              <select id="filter-section">
                <option value="">All</option>
              </select>
            </div>
          </div>
          <div class="filter-group" id="filter-tags-group" style="display:none">
            <label>Tags</label>
            <div class="filter-tags" id="filter-tags"></div>
          </div>
        </div>
      </details>

      <div class="search-toolbar">
        <div class="search-sort">
          <button class="sort-btn active" data-sort="relevance" type="button">Relevance</button>
          <button class="sort-btn" data-sort="date" type="button">Date</button>
        </div>
        <div class="search-stats" id="search-stats"></div>
      </div>

      <div id="search-results"></div>
    </div>

    <!-- MiniSearch from CDN (~10KB gzipped) -->
    <script src="https://cdn.jsdelivr.net/npm/minisearch@7.1.0/dist/umd/index.min.js"></script>
    <script src="/js/search.js"></script>
    <script>
      (function() {
        var statsEl = document.getElementById('search-stats');
        var search = new OSGSearch({
          inputSelector: '#search-input',
          resultsSelector: '#search-results',
          maxResults: 50,
          minChars: 2,
          onResults: function(shown, total) {
            if (shown > 0) {
              var text = shown + ' result' + (shown !== 1 ? 's' : '');
              if (shown < total) text += ' (filtered from ' + total + ')';
              statsEl.textContent = text;
            } else {
              statsEl.textContent = '';
            }
          }
        });

        search.ready.then(function() {
          // Populate section dropdown
          var sectionSelect = document.getElementById('filter-section');
          var sectionGroup = document.getElementById('filter-section-group');
          var sections = search.getAllSections();
          if (sections.length === 0) {
            sectionGroup.style.display = 'none';
          } else {
            sections.forEach(function(s) {
              var opt = document.createElement('option');
              opt.value = s;
              opt.textContent = s.charAt(0).toUpperCase() + s.slice(1);
              sectionSelect.appendChild(opt);
            });
          }

          // Populate tag pills
          var tagsContainer = document.getElementById('filter-tags');
          var tagsGroup = document.getElementById('filter-tags-group');
          var tags = search.getAllTags();
          if (tags.length > 0) {
            tagsGroup.style.display = '';
            tags.forEach(function(tag) {
              var btn = document.createElement('button');
              btn.className = 'filter-tag';
              btn.textContent = tag;
              btn.type = 'button';
              btn.addEventListener('click', function() {
                btn.classList.toggle('active');
                updateFilters();
              });
              tagsContainer.appendChild(btn);
            });
          }

          // Sort toggle
          var sortBtns = document.querySelectorAll('.sort-btn');
          sortBtns.forEach(function(btn) {
            btn.addEventListener('click', function() {
              sortBtns.forEach(function(b) { b.classList.remove('active'); });
              btn.classList.add('active');
              updateFilters();
            });
          });

          // Date and section change listeners
          document.getElementById('filter-date-from').addEventListener('change', updateFilters);
          document.getElementById('filter-date-to').addEventListener('change', updateFilters);
          sectionSelect.addEventListener('change', updateFilters);

          function updateFilters() {
            var activeTags = [];
            tagsContainer.querySelectorAll('.filter-tag.active').forEach(function(b) {
              activeTags.push(b.textContent.toLowerCase());
            });
            var activeSort = document.querySelector('.sort-btn.active');
            search.setFilters({
              dateFrom: document.getElementById('filter-date-from').value,
              dateTo: document.getElementById('filter-date-to').value,
              section: sectionSelect.value,
              tags: activeTags,
              sortBy: activeSort ? activeSort.dataset.sort : 'relevance'
            });
          }
        });
      })();
    </script>
  </body>
</html>"##.to_string()
}

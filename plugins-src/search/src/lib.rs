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

    let _ = write_search(&event.payload);
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

        // Strip HTML for plain text search index
        let content_text = strip_html(content_html);

        items.push(json!({
            "id": permalink,
            "title": title,
            "summary": summary,
            "content": content_text,
            "permalink": permalink,
            "date": date,
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
    this.miniSearch = null;
    this.documents = [];
    this.init();
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

      // Initialize MiniSearch with indexed fields
      this.miniSearch = new MiniSearch({
        fields: ['title', 'summary', 'content', 'tags'],
        storeFields: ['title', 'permalink', 'date', 'summary'],
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
        summary: doc.summary || ''
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

    this.input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        this.hide();
        this.input.blur();
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        this.navigateResults(e.key === 'ArrowDown' ? 1 : -1);
      }
      if (e.key === 'Enter') {
        const selected = this.results.querySelector('a:focus');
        if (selected) window.location = selected.href;
      }
    });
  }

  search(query) {
    if (!this.miniSearch || query.length < this.minChars) {
      this.hide();
      return;
    }

    try {
      const results = this.miniSearch.search(query, { prefix: true, fuzzy: 0.2 })
        .slice(0, this.maxResults);
      this.render(results, query);
    } catch (err) {
      // If search fails (e.g., empty query), show no results
      this.results.innerHTML = `<div class="search-no-results">No results for "${this.escapeHtml(query)}"</div>`;
      this.show();
    }
  }

  render(results, query) {
    if (results.length === 0) {
      this.results.innerHTML = `<div class="search-no-results">No results for "${this.escapeHtml(query)}"</div>`;
      this.show();
      return;
    }

    const html = results.map(r => {
      const doc = this.documents.find(d => d.permalink === r.id) || {};
      const excerpt = this.getExcerpt(doc, query);
      return `<li class="search-result">
        <a href="${r.id}">
          <span class="search-result-title">${this.escapeHtml(r.title || doc.title || '')}</span>
          <span class="search-result-date">${r.date || doc.date || ''}</span>
          <span class="search-result-excerpt">${excerpt}</span>
        </a>
      </li>`;
    }).join('');

    this.results.innerHTML = `<ul class="search-results-list">${html}</ul>`;
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
      excerpt = (start > 0 ? '…' : '') + content.slice(start, end) + (end < content.length ? '…' : '');
    } else if (doc.summary) {
      excerpt = doc.summary.slice(0, 120) + (doc.summary.length > 120 ? '…' : '');
    } else {
      excerpt = content.slice(0, 120) + (content.length > 120 ? '…' : '');
    }

    return this.highlight(excerpt, query);
  }

  highlight(text, query) {
    const terms = query.toLowerCase().split(/\s+/).filter(t => t);
    let result = this.escapeHtml(text);
    for (const term of terms) {
      const regex = new RegExp(`(${this.escapeRegex(term)})`, 'gi');
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

  navigateResults(direction) {
    const links = this.results.querySelectorAll('a');
    if (links.length === 0) return;

    const current = this.results.querySelector('a:focus');
    let index = -1;
    for (let i = 0; i < links.length; i++) {
      if (links[i] === current) {
        index = i;
        break;
      }
    }

    const nextIndex = direction > 0
      ? (index + 1) % links.length
      : (index - 1 + links.length) % links.length;

    links[nextIndex].focus();
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
        --nord0: #2e3440; --nord1: #3b4252; --nord4: #d8dee9; --nord6: #eceff4;
        --nord8: #88c0d0; --nord10: #5e81ac; --nord13: #ebcb8b;
      }
      @media (prefers-color-scheme: light) {
        :root { --bg: var(--nord6); --text: var(--nord0); --muted: var(--nord1); --accent: var(--nord10); }
      }
      @media (prefers-color-scheme: dark) {
        :root { --bg: var(--nord0); --text: var(--nord4); --muted: var(--nord1); --accent: var(--nord8); }
      }
      body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); margin: 0; padding: 2rem; min-height: 100vh; }
      .search-container { max-width: 800px; margin: 0 auto; }
      h1 { margin-bottom: 1.5rem; }
      #search-input { width: 100%; padding: 1rem 1.25rem; font-size: 1.25rem; border: 2px solid var(--muted); border-radius: 8px; background: var(--bg); color: var(--text); box-sizing: border-box; outline: none; transition: border-color 0.2s; }
      #search-input:focus { border-color: var(--accent); }
      #search-input::placeholder { color: var(--muted); }
      .search-stats { margin: 1rem 0; color: var(--muted); font-size: 0.9rem; }
      #search-results ul { list-style: none; padding: 0; margin: 0; }
      .search-result { margin: 1rem 0; padding: 1rem; border-radius: 8px; background: var(--muted); transition: transform 0.15s; }
      .search-result:hover { transform: translateX(4px); }
      .search-result a { text-decoration: none; color: inherit; display: block; }
      .search-result-title { font-size: 1.1rem; font-weight: 600; color: var(--accent); display: block; }
      .search-result-date { font-size: 0.85rem; color: var(--muted); display: block; margin: 0.25rem 0; }
      .search-result-excerpt { font-size: 0.95rem; line-height: 1.5; margin-top: 0.5rem; }
      mark { background: var(--nord13); color: var(--nord0); padding: 0 0.2em; border-radius: 2px; }
      .search-no-results { padding: 2rem; text-align: center; color: var(--muted); }
    </style>
  </head>
  <body>
    <div class="search-container">
      <h1>Search</h1>
      <input id="search-input" type="search" placeholder="Type to search..." autocomplete="off" />
      <div class="search-stats" id="search-stats"></div>
      <div id="search-results"></div>
    </div>
    <!-- MiniSearch from CDN (~10KB gzipped) -->
    <script src="https://cdn.jsdelivr.net/npm/minisearch@7.1.0/dist/umd/index.min.js"></script>
    <script src="/js/search.js"></script>
    <script>
      const search = new OSGSearch({ inputSelector: '#search-input', resultsSelector: '#search-results', maxResults: 50, minChars: 2 });
      const statsEl = document.getElementById('search-stats');
      const inputEl = document.getElementById('search-input');
      inputEl.addEventListener('input', () => {
        const count = document.querySelectorAll('.search-result').length;
        statsEl.textContent = inputEl.value.length >= 2 && count > 0 ? count + ' result' + (count !== 1 ? 's' : '') : '';
      });
    </script>
  </body>
</html>"##.to_string()
}

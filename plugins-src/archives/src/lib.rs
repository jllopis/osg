use serde::Deserialize;
use serde_json::Value;
use std::collections::BTreeMap;
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

    if let Err(e) = write_archives(&event.payload) {
        eprintln!("osg-archives plugin: {}", e);
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn plugin_info() -> u64 {
    let info = r#"{"name":"archives","version":"0.1.0","description":"Chronological archive pages grouped by year and month","author":"jllopis","hooks":["build.finished"]}"#;
    let bytes = info.as_bytes();
    let ptr = alloc(bytes.len() as i32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, bytes.len());
    ((ptr as u64) << 32) | (bytes.len() as u64)
}

/// A post entry with the fields we need for archive display.
struct ArchiveEntry {
    title: String,
    permalink: String,
    date: String, // YYYY-MM-DD or similar
    summary: String,
    year: String,
    month: String,
    month_name: String,
}

fn write_archives(payload: &Value) -> Result<(), String> {
    let config = payload.get("config").ok_or("no config")?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .unwrap_or("public");
    let site_title = config
        .get("site_title")
        .and_then(|v| v.as_str())
        .unwrap_or("Site");
    let base_url = config
        .get("base_url")
        .and_then(|v| v.as_str())
        .unwrap_or("");

    let pages = payload
        .get("site")
        .and_then(|v| v.get("pages"))
        .and_then(|v| v.as_array())
        .ok_or("no pages")?;

    // Collect non-draft, non-menu posts with valid dates
    let mut entries: Vec<ArchiveEntry> = Vec::new();
    for page in pages {
        let is_menu = page.get("menu").and_then(|v| v.as_bool()).unwrap_or(false);
        let is_draft = page.get("draft").and_then(|v| v.as_bool()).unwrap_or(false);
        if is_menu || is_draft {
            continue;
        }

        let title = page
            .get("title")
            .and_then(|v| v.as_str())
            .unwrap_or("Untitled");
        let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
        let date = page.get("date").and_then(|v| v.as_str()).unwrap_or("");
        let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");

        if date.is_empty() {
            continue;
        }

        // Parse year and month from date (expected: YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS...)
        let (year, month, month_name) = parse_year_month(date);
        if year.is_empty() {
            continue;
        }

        entries.push(ArchiveEntry {
            title: title.to_string(),
            permalink: permalink.to_string(),
            date: date.to_string(),
            summary: summary.to_string(),
            year,
            month,
            month_name,
        });
    }

    // Sort by date descending
    entries.sort_by(|a, b| b.date.cmp(&a.date));

    // Group by year -> month (BTreeMap for sorted keys, reversed)
    // We use String keys like "2025" and "01" for ordering
    let mut by_year: BTreeMap<String, BTreeMap<String, Vec<&ArchiveEntry>>> = BTreeMap::new();
    for entry in &entries {
        by_year
            .entry(entry.year.clone())
            .or_default()
            .entry(entry.month.clone())
            .or_default()
            .push(entry);
    }

    let base = base_url.trim_end_matches('/');

    // Create /archive/ directory
    let archive_dir = Path::new(public_dir).join("archive");
    fs::create_dir_all(&archive_dir).map_err(|e| format!("mkdir {:?}: {}", archive_dir, e))?;

    // Generate main archive page (/archive/index.html)
    let main_html = render_main_archive(site_title, base, &by_year, &entries);
    let main_path = archive_dir.join("index.html");
    fs::write(&main_path, main_html.as_bytes())
        .map_err(|e| format!("write {:?}: {}", main_path, e))?;

    // Generate per-year pages (/archive/YYYY/index.html)
    for (year, months) in &by_year {
        let year_dir = archive_dir.join(year);
        fs::create_dir_all(&year_dir).map_err(|e| format!("mkdir {:?}: {}", year_dir, e))?;

        let year_html = render_year_archive(site_title, base, year, months);
        let year_path = year_dir.join("index.html");
        fs::write(&year_path, year_html.as_bytes())
            .map_err(|e| format!("write {:?}: {}", year_path, e))?;
    }

    Ok(())
}

fn parse_year_month(date: &str) -> (String, String, String) {
    // Handle both "2025-01-15" and "2025-01-15T10:30:00Z" formats
    let parts: Vec<&str> = date
        .split(|c: char| c == '-' || c == 'T' || c == ' ')
        .collect();
    if parts.len() < 2 {
        return (String::new(), String::new(), String::new());
    }

    let year = parts[0].to_string();
    let month = parts[1].to_string();
    let month_name = match month.as_str() {
        "01" => "January",
        "02" => "February",
        "03" => "March",
        "04" => "April",
        "05" => "May",
        "06" => "June",
        "07" => "July",
        "08" => "August",
        "09" => "September",
        "10" => "October",
        "11" => "November",
        "12" => "December",
        _ => "Unknown",
    }
    .to_string();

    (year, month, month_name)
}

fn render_main_archive(
    site_title: &str,
    base: &str,
    by_year: &BTreeMap<String, BTreeMap<String, Vec<&ArchiveEntry>>>,
    entries: &[ArchiveEntry],
) -> String {
    let mut html = String::new();
    html.push_str(&archive_html_head(
        &format!("Archive | {}", site_title),
        base,
    ));

    html.push_str("<body>\n<div class=\"archive-container\">\n");
    html.push_str(&format!(
        "<header class=\"archive-header\">\n  <a href=\"{}\" class=\"archive-home\">&larr; Home</a>\n  <h1>Archive</h1>\n  <p class=\"archive-count\">{} post{}</p>\n</header>\n",
        if base.is_empty() { "/" } else { base },
        entries.len(),
        if entries.len() == 1 { "" } else { "s" }
    ));

    // Year navigation
    html.push_str("<nav class=\"archive-years\" aria-label=\"Archive years\">\n");
    for year in by_year.keys().rev() {
        let count: usize = by_year[year].values().map(|v| v.len()).sum();
        html.push_str(&format!(
            "  <a href=\"/archive/{}\">{} <span class=\"badge\">{}</span></a>\n",
            year, year, count
        ));
    }
    html.push_str("</nav>\n\n");

    // Full listing by year and month
    for year in by_year.keys().rev() {
        let months = &by_year[year];
        html.push_str(&format!(
            "<section class=\"archive-year\" id=\"year-{}\">\n",
            year
        ));
        html.push_str(&format!(
            "  <h2><a href=\"/archive/{}\">{}</a></h2>\n",
            year, year
        ));

        for month_key in months.keys().rev() {
            let posts = &months[month_key];
            if let Some(first) = posts.first() {
                html.push_str(&format!(
                    "  <h3 class=\"archive-month\">{}</h3>\n",
                    first.month_name
                ));
            }
            html.push_str("  <ul class=\"archive-list\">\n");
            for post in posts {
                let display_date = &post.date[..10.min(post.date.len())];
                html.push_str(&format!(
                    "    <li>\n      <time datetime=\"{}\">{}</time>\n      <a href=\"{}\">{}</a>\n",
                    post.date,
                    display_date,
                    escape_html(&post.permalink),
                    escape_html(&post.title)
                ));
                if !post.summary.is_empty() {
                    html.push_str(&format!(
                        "      <span class=\"archive-summary\">{}</span>\n",
                        escape_html(&post.summary)
                    ));
                }
                html.push_str("    </li>\n");
            }
            html.push_str("  </ul>\n");
        }

        html.push_str("</section>\n\n");
    }

    html.push_str("</div>\n</body>\n</html>");
    html
}

fn render_year_archive(
    site_title: &str,
    base: &str,
    year: &str,
    months: &BTreeMap<String, Vec<&ArchiveEntry>>,
) -> String {
    let total: usize = months.values().map(|v| v.len()).sum();
    let mut html = String::new();
    html.push_str(&archive_html_head(
        &format!("{} Archive | {}", year, site_title),
        base,
    ));

    html.push_str("<body>\n<div class=\"archive-container\">\n");
    html.push_str(&format!(
        "<header class=\"archive-header\">\n  <a href=\"/archive/\" class=\"archive-home\">&larr; All years</a>\n  <h1>{}</h1>\n  <p class=\"archive-count\">{} post{}</p>\n</header>\n\n",
        year,
        total,
        if total == 1 { "" } else { "s" }
    ));

    for month_key in months.keys().rev() {
        let posts = &months[month_key];
        if let Some(first) = posts.first() {
            html.push_str(&format!(
                "<section class=\"archive-month-section\">\n  <h2 class=\"archive-month\">{}</h2>\n",
                first.month_name
            ));
        }
        html.push_str("  <ul class=\"archive-list\">\n");
        for post in posts {
            let display_date = &post.date[..10.min(post.date.len())];
            html.push_str(&format!(
                "    <li>\n      <time datetime=\"{}\">{}</time>\n      <a href=\"{}\">{}</a>\n",
                post.date,
                display_date,
                escape_html(&post.permalink),
                escape_html(&post.title)
            ));
            if !post.summary.is_empty() {
                html.push_str(&format!(
                    "      <span class=\"archive-summary\">{}</span>\n",
                    escape_html(&post.summary)
                ));
            }
            html.push_str("    </li>\n");
        }
        html.push_str("  </ul>\n</section>\n\n");
    }

    html.push_str("</div>\n</body>\n</html>");
    html
}

fn archive_html_head(title: &str, base: &str) -> String {
    let base_tag = if base.is_empty() {
        String::new()
    } else {
        format!("  <base href=\"{}/\">\n", base)
    };

    format!(
        r#"<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
{base_tag}  <title>{title}</title>
  <style>
    :root {{
      --nord0: #2e3440; --nord1: #3b4252; --nord2: #434c5e; --nord3: #4c566a;
      --nord4: #d8dee9; --nord5: #e5e9f0; --nord6: #eceff4;
      --nord7: #8fbcbb; --nord8: #88c0d0; --nord9: #81a1c1; --nord10: #5e81ac;
      --nord11: #bf616a; --nord12: #d08770; --nord13: #ebcb8b;
      --nord14: #a3be8c; --nord15: #b48ead;
    }}
    @media (prefers-color-scheme: light) {{
      :root {{ --bg: var(--nord6); --bg-subtle: var(--nord5); --text: var(--nord0);
        --text-muted: var(--nord3); --accent: var(--nord10); --accent-hover: #4c6e96;
        --border: var(--nord4); --surface: #fff; }}
    }}
    @media (prefers-color-scheme: dark) {{
      :root {{ --bg: var(--nord0); --bg-subtle: var(--nord1); --text: var(--nord4);
        --text-muted: var(--nord3); --accent: var(--nord8); --accent-hover: var(--nord9);
        --border: var(--nord2); --surface: var(--nord1); }}
    }}
    *, *::before, *::after {{ box-sizing: border-box; margin: 0; padding: 0; }}
    body {{
      font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: var(--bg); color: var(--text); line-height: 1.6;
      min-height: 100vh;
    }}
    .archive-container {{ max-width: 800px; margin: 0 auto; padding: 2rem 1.5rem; }}
    .archive-header {{ margin-bottom: 2rem; }}
    .archive-home {{
      color: var(--accent); text-decoration: none; font-size: 0.9rem;
      display: inline-block; margin-bottom: 0.5rem;
    }}
    .archive-home:hover {{ color: var(--accent-hover); }}
    .archive-header h1 {{ font-size: 2rem; font-weight: 700; letter-spacing: -0.02em; }}
    .archive-count {{ color: var(--text-muted); font-size: 0.9rem; margin-top: 0.25rem; }}
    .archive-years {{
      display: flex; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 2rem;
      padding: 1rem; background: var(--surface); border-radius: 8px;
      border: 1px solid var(--border);
    }}
    .archive-years a {{
      color: var(--accent); text-decoration: none; font-weight: 600;
      padding: 0.25rem 0.5rem; border-radius: 4px; transition: background 0.15s;
    }}
    .archive-years a:hover {{ background: var(--bg-subtle); }}
    .badge {{
      font-size: 0.75rem; font-weight: 400; color: var(--text-muted);
      vertical-align: super;
    }}
    .archive-year {{ margin-bottom: 2rem; }}
    .archive-year h2 {{
      font-size: 1.5rem; font-weight: 700; margin-bottom: 1rem;
      border-bottom: 2px solid var(--accent); padding-bottom: 0.5rem;
    }}
    .archive-year h2 a {{ color: inherit; text-decoration: none; }}
    .archive-year h2 a:hover {{ color: var(--accent); }}
    .archive-month, h2.archive-month {{
      font-size: 1.1rem; font-weight: 600; color: var(--accent);
      margin: 1.25rem 0 0.5rem; padding-left: 0.25rem;
    }}
    .archive-list {{ list-style: none; padding: 0; }}
    .archive-list li {{
      display: grid; grid-template-columns: auto 1fr; grid-template-rows: auto auto;
      gap: 0 1rem; padding: 0.5rem 0.5rem; border-radius: 6px;
      transition: background 0.15s;
    }}
    .archive-list li:hover {{ background: var(--bg-subtle); }}
    .archive-list time {{
      font-family: "JetBrains Mono", ui-monospace, monospace;
      font-size: 0.85rem; color: var(--text-muted); white-space: nowrap;
      grid-row: 1; grid-column: 1; padding-top: 0.1rem;
    }}
    .archive-list a {{
      color: var(--text); text-decoration: none; font-weight: 500;
      grid-row: 1; grid-column: 2;
    }}
    .archive-list a:hover {{ color: var(--accent); }}
    .archive-summary {{
      font-size: 0.85rem; color: var(--text-muted); grid-row: 2; grid-column: 2;
      display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
      overflow: hidden;
    }}
    .archive-month-section {{ margin-bottom: 1.5rem; }}
    @media (max-width: 600px) {{
      .archive-list li {{ grid-template-columns: 1fr; }}
      .archive-list time {{ grid-column: 1; margin-bottom: 0.125rem; }}
      .archive-list a {{ grid-column: 1; grid-row: 2; }}
      .archive-summary {{ grid-column: 1; grid-row: 3; }}
    }}
  </style>
</head>
"#,
        title = escape_html(title),
        base_tag = base_tag,
    )
}

fn escape_html(input: &str) -> String {
    input
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&#39;")
}

use serde::Deserialize;
use serde_json::Value;
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

    if let Err(e) = write_llmstxt(&event.payload) {
        eprintln!("osg-llmstxt plugin: {}", e);
    }
    0
}

#[no_mangle]
pub unsafe extern "C" fn plugin_info() -> u64 {
    let info = r#"{"name":"llmstxt","version":"0.1.0","description":"Generate llms.txt and llms-full.txt for LLM consumption","author":"jllopis","hooks":["build.finished"]}"#;
    let bytes = info.as_bytes();
    let ptr = alloc(bytes.len() as i32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, bytes.len());
    ((ptr as u64) << 32) | (bytes.len() as u64)
}

/// Strip HTML tags and normalize whitespace for plain text output.
fn strip_html(html: &str) -> String {
    let mut result = String::with_capacity(html.len());
    let mut in_tag = false;
    let mut prev_space = false;

    for c in html.chars() {
        if c == '<' {
            in_tag = true;
            continue;
        }
        if c == '>' {
            in_tag = false;
            // Insert a space after block-level tags to preserve word boundaries
            if !prev_space {
                result.push(' ');
                prev_space = true;
            }
            continue;
        }
        if !in_tag {
            if c.is_whitespace() {
                if !prev_space {
                    result.push(' ');
                    prev_space = true;
                }
            } else {
                result.push(c);
                prev_space = false;
            }
        }
    }

    result.trim().to_string()
}

/// Decode common HTML entities to plain text.
fn decode_entities(text: &str) -> String {
    text.replace("&amp;", "&")
        .replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&quot;", "\"")
        .replace("&#39;", "'")
        .replace("&apos;", "'")
        .replace("&nbsp;", " ")
}

fn write_llmstxt(payload: &Value) -> Result<(), String> {
    let config = payload.get("config").ok_or("no config")?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .unwrap_or("public");
    let base_url = config
        .get("base_url")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let site_title = config
        .get("site_title")
        .and_then(|v| v.as_str())
        .unwrap_or("Site");
    let site_description = config
        .get("site_description")
        .and_then(|v| v.as_str())
        .unwrap_or("");

    let pages = payload
        .get("site")
        .and_then(|v| v.get("pages"))
        .and_then(|v| v.as_array())
        .ok_or("no pages")?;

    // Separate menu pages (standalone) from regular posts
    let mut menu_pages: Vec<&Value> = Vec::new();
    let mut posts: Vec<&Value> = Vec::new();

    for page in pages {
        let is_menu = page.get("menu").and_then(|v| v.as_bool()).unwrap_or(false);
        let is_draft = page.get("draft").and_then(|v| v.as_bool()).unwrap_or(false);
        if is_draft {
            continue;
        }
        if is_menu {
            menu_pages.push(page);
        } else {
            posts.push(page);
        }
    }

    // Sort posts by date descending
    posts.sort_by(|a, b| {
        let da = a.get("date").and_then(|v| v.as_str()).unwrap_or("");
        let db = b.get("date").and_then(|v| v.as_str()).unwrap_or("");
        db.cmp(da)
    });

    let base = base_url.trim_end_matches('/');

    // --- llms.txt (summary version) ---
    let mut txt = String::new();

    // Header: site title and description
    txt.push_str(&format!("# {}\n\n", site_title));
    if !site_description.is_empty() {
        txt.push_str(&format!("> {}\n\n", site_description));
    }

    // Standalone pages section
    if !menu_pages.is_empty() {
        txt.push_str("## Pages\n\n");
        for page in &menu_pages {
            let title = page
                .get("title")
                .and_then(|v| v.as_str())
                .unwrap_or("Untitled");
            let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
            let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");
            let url = format_url(base, permalink);

            if summary.is_empty() {
                txt.push_str(&format!("- [{}]({})\n", title, url));
            } else {
                txt.push_str(&format!("- [{}]({}): {}\n", title, url, summary));
            }
        }
        txt.push('\n');
    }

    // Posts section
    if !posts.is_empty() {
        txt.push_str("## Posts\n\n");
        for page in &posts {
            let title = page
                .get("title")
                .and_then(|v| v.as_str())
                .unwrap_or("Untitled");
            let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
            let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");
            let url = format_url(base, permalink);

            if summary.is_empty() {
                txt.push_str(&format!("- [{}]({})\n", title, url));
            } else {
                txt.push_str(&format!("- [{}]({}): {}\n", title, url, summary));
            }
        }
        txt.push('\n');
    }

    let txt_path = Path::new(public_dir).join("llms.txt");
    fs::write(&txt_path, txt.as_bytes()).map_err(|e| format!("write {:?}: {}", txt_path, e))?;

    // --- llms-full.txt (full content version) ---
    let mut full = String::new();

    // Header
    full.push_str(&format!("# {}\n\n", site_title));
    if !site_description.is_empty() {
        full.push_str(&format!("> {}\n\n", site_description));
    }

    // Standalone pages with full content
    if !menu_pages.is_empty() {
        full.push_str("## Pages\n\n");
        for page in &menu_pages {
            append_full_page(&mut full, page, base);
        }
    }

    // Posts with full content
    if !posts.is_empty() {
        full.push_str("## Posts\n\n");
        for page in &posts {
            append_full_page(&mut full, page, base);
        }
    }

    let full_path = Path::new(public_dir).join("llms-full.txt");
    fs::write(&full_path, full.as_bytes()).map_err(|e| format!("write {:?}: {}", full_path, e))?;

    Ok(())
}

fn append_full_page(out: &mut String, page: &Value, base: &str) {
    let title = page
        .get("title")
        .and_then(|v| v.as_str())
        .unwrap_or("Untitled");
    let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
    let date = page.get("date").and_then(|v| v.as_str()).unwrap_or("");
    let content_html = page.get("content").and_then(|v| v.as_str()).unwrap_or("");
    let url = format_url(base, permalink);

    out.push_str(&format!("### {}\n\n", title));
    if !date.is_empty() {
        out.push_str(&format!("Date: {}\n", date));
    }
    out.push_str(&format!("URL: {}\n\n", url));

    if !content_html.is_empty() {
        let plain = decode_entities(&strip_html(content_html));
        out.push_str(&plain);
        out.push('\n');
    }
    out.push_str("\n---\n\n");
}

/// Build a full URL from base and permalink.
/// If permalink already starts with http(s), return as-is.
/// Otherwise combine base_url + permalink.
fn format_url(base: &str, permalink: &str) -> String {
    if permalink.starts_with("http://") || permalink.starts_with("https://") {
        return permalink.to_string();
    }
    if base.is_empty() {
        return permalink.to_string();
    }
    let trimmed = permalink.trim_start_matches('/');
    format!("{}/{}", base, trimmed)
}

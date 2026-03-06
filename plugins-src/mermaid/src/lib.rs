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

/// Mermaid CDN version to use.
const MERMAID_VERSION: &str = "11.4.1";

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

    match event.event_type.as_str() {
        "content.transform" => handle_content_transform(&event.payload),
        "build.finished" => {
            let _ = write_mermaid_loader(&event.payload);
            0
        }
        _ => 0,
    }
}

#[no_mangle]
pub unsafe extern "C" fn plugin_info() -> u64 {
    let info = r#"{"name":"mermaid","version":"0.1.0","description":"Client-side Mermaid diagram rendering via CDN","author":"jllopis","hooks":["content.transform","build.finished"]}"#;
    let bytes = info.as_bytes();
    let ptr = alloc(bytes.len() as i32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, bytes.len());
    ((ptr as u64) << 32) | (bytes.len() as u64)
}

/// Transform markdown: replace ```mermaid code blocks with HTML <pre class="mermaid">
/// blocks that mermaid.js will render client-side.
///
/// Returns the packed ptr|len of a JSON response if any transformation was made,
/// or 0 if nothing changed.
fn handle_content_transform(payload: &Value) -> u64 {
    let body = match payload
        .get("page")
        .and_then(|p| p.get("body_markdown"))
        .and_then(|v| v.as_str())
    {
        Some(b) => b,
        None => return 0,
    };

    // Quick check: does this page even have mermaid blocks?
    if !body.contains("```mermaid") {
        return 0;
    }

    let transformed = rewrite_mermaid_blocks(body);
    if transformed == body {
        return 0;
    }

    // Return the modified body_markdown
    let response = json!({
        "page": {
            "body_markdown": transformed
        }
    });

    let response_bytes = response.to_string().into_bytes();
    let len = response_bytes.len();
    unsafe {
        let ptr = alloc(len as i32);
        std::ptr::copy_nonoverlapping(response_bytes.as_ptr(), ptr as *mut u8, len);
        ((ptr as u64) << 32) | (len as u64)
    }
}

/// Rewrite ```mermaid ... ``` blocks into raw HTML blocks that Goldmark will
/// pass through as-is. We use a blank line + raw HTML block pattern:
///
///   <pre class="mermaid">
///   graph TD
///     A --> B
///   </pre>
///
/// Goldmark treats lines starting with certain HTML tags as HTML blocks,
/// so `<pre>` will be passed through without escaping.
fn rewrite_mermaid_blocks(input: &str) -> String {
    let mut output = String::with_capacity(input.len());
    let mut lines = input.lines().peekable();
    let mut modified = false;

    while let Some(line) = lines.next() {
        let trimmed = line.trim();

        // Detect opening ```mermaid fence (with optional spaces before ```)
        if trimmed == "```mermaid" || trimmed.starts_with("```mermaid ") {
            modified = true;
            // Collect everything until the closing ```
            let mut diagram = String::new();
            let mut found_close = false;
            while let Some(inner) = lines.next() {
                if inner.trim() == "```" {
                    found_close = true;
                    break;
                }
                if !diagram.is_empty() {
                    diagram.push('\n');
                }
                diagram.push_str(inner);
            }

            if !found_close {
                // Malformed block: no closing fence. Output as-is.
                output.push_str(line);
                output.push('\n');
                output.push_str(&diagram);
                continue;
            }

            // Emit as raw HTML block (blank line before and after for Goldmark)
            output.push_str("\n<pre class=\"mermaid\">\n");
            // HTML-escape the diagram content to prevent XSS
            output.push_str(&escape_html(&diagram));
            output.push_str("\n</pre>\n\n");
        } else {
            output.push_str(line);
            output.push('\n');
        }
    }

    if modified {
        // Remove trailing extra newline if present
        while output.ends_with("\n\n\n") {
            output.pop();
        }
        output
    } else {
        input.to_string()
    }
}

/// Minimal HTML escaping for diagram content inside <pre> tags.
fn escape_html(input: &str) -> String {
    input
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
}

/// Write the mermaid.js loader script to public/js/mermaid-init.js.
/// This script:
/// 1. Checks if any <pre class="mermaid"> elements exist on the page
/// 2. If so, dynamically loads mermaid.js from CDN and initializes it
/// 3. Respects prefers-color-scheme for theme selection
fn write_mermaid_loader(payload: &Value) -> Result<(), String> {
    let config = payload.get("config").ok_or("no config")?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .unwrap_or("public");

    let js_dir = Path::new(public_dir).join("js");
    fs::create_dir_all(&js_dir).map_err(|e| format!("mkdir {:?}: {}", js_dir, e))?;

    let js_path = js_dir.join("mermaid-init.js");
    fs::write(&js_path, mermaid_init_js()).map_err(|e| format!("write {:?}: {}", js_path, e))?;

    Ok(())
}

fn mermaid_init_js() -> String {
    format!(
        r#"(function() {{
  'use strict';

  // Only load mermaid if there are diagrams on the page
  var diagrams = document.querySelectorAll('pre.mermaid');
  if (diagrams.length === 0) return;

  // Determine theme from color scheme
  var isDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  var theme = isDark ? 'dark' : 'default';

  // Check if the page has an explicit color scheme
  var root = document.documentElement;
  if (root.dataset.colorScheme === 'dark') theme = 'dark';
  else if (root.dataset.colorScheme === 'light') theme = 'default';

  // Decode HTML entities in diagram content before mermaid processes them
  diagrams.forEach(function(pre) {{
    var text = pre.textContent;
    // textContent already decodes entities, so we just need to set it back
    // to ensure mermaid reads the raw diagram syntax
    pre.textContent = text;
  }});

  // Dynamically load mermaid from CDN
  var script = document.createElement('script');
  script.src = 'https://cdn.jsdelivr.net/npm/mermaid@{version}/dist/mermaid.min.js';
  script.onload = function() {{
    mermaid.initialize({{
      startOnLoad: true,
      theme: theme,
      securityLevel: 'strict',
      fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    }});
  }};
  document.head.appendChild(script);
}})();
"#,
        version = MERMAID_VERSION
    )
}

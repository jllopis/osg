use serde::Deserialize;
use serde_json::{json, Value};
use std::mem;

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
        "page.before_render" => handle_page_before_render(&event.payload),
        _ => 0,
    }
}

#[no_mangle]
pub unsafe extern "C" fn plugin_info() -> u64 {
    let info = r#"{"name":"mermaid","version":"0.2.0","description":"Client-side Mermaid diagram rendering via CDN","author":"jllopis","hooks":["content.transform","page.before_render"]}"#;
    let bytes = info.as_bytes();
    let ptr = alloc(bytes.len() as i32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, bytes.len());
    ((ptr as u64) << 32) | (bytes.len() as u64)
}

/// Transform markdown: replace ```mermaid code blocks with HTML <pre class="mermaid">
/// blocks that mermaid.js will render client-side.
fn handle_content_transform(payload: &Value) -> u64 {
    let body = match payload
        .get("page")
        .and_then(|p| p.get("body_markdown"))
        .and_then(|v| v.as_str())
    {
        Some(b) => b,
        None => return 0,
    };

    if !body.contains("```mermaid") {
        return 0;
    }

    let transformed = rewrite_mermaid_blocks(body);
    if transformed == body {
        return 0;
    }

    let response = json!({
        "payload": {
            "page": {
                "body_markdown": transformed
            }
        }
    });

    json_to_wasm(&response)
}

/// Inject an inline mermaid loader script at the end of page.content
/// for pages that contain mermaid diagrams.
fn handle_page_before_render(payload: &Value) -> u64 {
    let content = match payload
        .get("page")
        .and_then(|p| p.get("content"))
        .and_then(|v| v.as_str())
    {
        Some(c) => c,
        None => return 0,
    };

    if !content.contains("class=\"mermaid\"") {
        return 0;
    }

    let mut modified = String::with_capacity(content.len() + 1024);
    modified.push_str(content);
    modified.push_str("\n");
    modified.push_str(&mermaid_inline_script());

    let response = json!({
        "payload": {
            "page": {
                "content": modified
            }
        }
    });

    json_to_wasm(&response)
}

/// Serialize a JSON value into WASM memory and return packed ptr|len.
fn json_to_wasm(value: &Value) -> u64 {
    let bytes = value.to_string().into_bytes();
    let len = bytes.len();
    unsafe {
        let ptr = alloc(len as i32);
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, len);
        ((ptr as u64) << 32) | (len as u64)
    }
}

/// Rewrite ```mermaid ... ``` blocks into raw HTML blocks that Goldmark will
/// pass through as-is.
fn rewrite_mermaid_blocks(input: &str) -> String {
    let mut output = String::with_capacity(input.len());
    let mut lines = input.lines().peekable();
    let mut modified = false;

    while let Some(line) = lines.next() {
        let trimmed = line.trim();

        if trimmed == "```mermaid" || trimmed.starts_with("```mermaid ") {
            modified = true;
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
                output.push_str(line);
                output.push('\n');
                output.push_str(&diagram);
                continue;
            }

            output.push_str("\n<pre class=\"mermaid\">\n");
            output.push_str(&escape_html(&diagram));
            output.push_str("\n</pre>\n\n");
        } else {
            output.push_str(line);
            output.push('\n');
        }
    }

    if modified {
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

/// Inline script that loads mermaid from CDN and initializes it.
/// Only executes if <pre class="mermaid"> elements exist on the page.
fn mermaid_inline_script() -> String {
    format!(
        r#"<script>(function(){{
  var d=document.querySelectorAll('pre.mermaid');
  if(!d.length)return;
  var isDark=window.matchMedia&&window.matchMedia('(prefers-color-scheme:dark)').matches;
  var root=document.documentElement;
  var theme=root.dataset.colorScheme==='dark'?'dark':root.dataset.colorScheme==='light'?'default':isDark?'dark':'default';
  d.forEach(function(pre){{pre.textContent=pre.textContent}});
  var s=document.createElement('script');
  s.src='https://cdn.jsdelivr.net/npm/mermaid@{version}/dist/mermaid.min.js';
  s.onload=function(){{mermaid.initialize({{startOnLoad:true,theme:theme,securityLevel:'strict'}})}};
  document.head.appendChild(s);
}})();</script>"#,
        version = MERMAID_VERSION
    )
}

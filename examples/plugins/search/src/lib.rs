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

    if let Err(_) = write_search(&event.payload) {
        return 0;
    }

    0
}

fn write_search(payload: &Value) -> Result<(), ()> {
    let config = payload.get("config").ok_or(())?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .unwrap_or("public");

    let pages = payload
        .get("site")
        .and_then(|v| v.get("pages"))
        .and_then(|v| v.as_array())
        .ok_or(())?;

    let mut items = Vec::new();
    for page in pages {
        let title = page.get("title").and_then(|v| v.as_str()).unwrap_or("");
        let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");
        let permalink = page.get("permalink").and_then(|v| v.as_str()).unwrap_or("");
        let date = page.get("date").and_then(|v| v.as_str()).unwrap_or("");
        let taxonomies = page.get("taxonomies").unwrap_or(&Value::Null);

        items.push(json!({
            "title": title,
            "summary": summary,
            "permalink": permalink,
            "date": date,
            "taxonomies": taxonomies
        }));
    }

    let search = json!({
        "items": items
    });

    let search_path = Path::new(public_dir).join("search.json");
    fs::write(&search_path, search.to_string()).map_err(|_| ())?;

    let html_path = Path::new(public_dir).join("search").join("index.html");
    if let Some(dir) = html_path.parent() {
        fs::create_dir_all(dir).map_err(|_| ())?;
    }
    fs::write(&html_path, search_html()).map_err(|_| ())?;
    Ok(())
}

fn search_html() -> String {
    let html = r#"<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Search</title>
    <style>
      body { font-family: sans-serif; margin: 2rem; max-width: 800px; }
      input { width: 100%; padding: 0.75rem; font-size: 1rem; }
      ul { list-style: none; padding: 0; }
      li { margin: 1rem 0; }
      .meta { color: #666; font-size: 0.85rem; }
    </style>
  </head>
  <body>
    <h1>Search</h1>
    <input id="q" type="search" placeholder="Type to search..." />
    <ul id="results"></ul>
    <script>
      const input = document.getElementById("q");
      const list = document.getElementById("results");
      let data = [];
      fetch("/search.json")
        .then(r => r.json())
        .then(json => { data = json.items || []; render(); });
      input.addEventListener("input", render);
      function normalize(item) {
        const taxo = item.taxonomies || {};
        const tags = Object.values(taxo).flat().join(" ");
        return `${item.title} ${item.summary} ${tags}`.toLowerCase();
      }
      function render() {
        const q = input.value.trim().toLowerCase();
        const items = q === "" ? data : data.filter(item => normalize(item).includes(q));
        list.innerHTML = "";
        for (const item of items) {
          const li = document.createElement("li");
          const a = document.createElement("a");
          a.href = item.permalink || "#";
          a.textContent = item.title || "Untitled";
          const meta = document.createElement("div");
          meta.className = "meta";
          meta.textContent = item.date || "";
          li.appendChild(a);
          li.appendChild(meta);
          if (item.summary) {
            const p = document.createElement("p");
            p.textContent = item.summary;
            li.appendChild(p);
          }
          list.appendChild(li);
        }
      }
    </script>
  </body>
</html>"#;
    html.to_string()
}

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

    if let Err(_) = write_rss(&event.payload) {
        return 0;
    }

    0
}

fn write_rss(payload: &Value) -> Result<(), ()> {
    let config = payload.get("config").ok_or(())?;
    let public_dir = config
        .get("public_dir")
        .and_then(|v| v.as_str())
        .ok_or(())?;
    let base_url = config
        .get("base_url")
        .and_then(|v| v.as_str())
        .unwrap_or("");

    let pages = payload
        .get("site")
        .and_then(|v| v.get("pages"))
        .and_then(|v| v.as_array())
        .ok_or(())?;

    let mut rss = String::from(
        "<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n<rss version=\"2.0\">\n<channel>\n",
    );
    rss.push_str("  <title>OSG Feed</title>\n");
    if !base_url.is_empty() {
        rss.push_str(&format!(
            "  <link>{}</link>\n",
            base_url.trim_end_matches('/')
        ));
    }
    rss.push_str("  <description>Latest content</description>\n");

    for page in pages {
        let title = page.get("title").and_then(|v| v.as_str()).unwrap_or("Untitled");
        let link = page
            .get("permalink")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let summary = page.get("summary").and_then(|v| v.as_str()).unwrap_or("");
        let date = page.get("date").and_then(|v| v.as_str()).unwrap_or("");

        rss.push_str("  <item>\n");
        rss.push_str(&format!("    <title>{}</title>\n", escape_xml(title)));
        if !link.is_empty() {
            rss.push_str(&format!("    <link>{}</link>\n", escape_xml(link)));
        }
        if !summary.is_empty() {
            rss.push_str(&format!(
                "    <description>{}</description>\n",
                escape_xml(summary)
            ));
        }
        if !date.is_empty() {
            rss.push_str(&format!("    <pubDate>{}</pubDate>\n", escape_xml(date)));
        }
        rss.push_str("  </item>\n");
    }

    rss.push_str("</channel>\n</rss>");

    let path = Path::new(public_dir).join("rss.xml");
    fs::write(&path, rss).map_err(|_| ())?;
    Ok(())
}

fn escape_xml(input: &str) -> String {
    input
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

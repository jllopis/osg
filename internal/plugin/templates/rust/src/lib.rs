use serde::Deserialize;
use serde_json::Value;
use std::mem;

// ---- WASM ABI: memory management ----

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

// ---- Plugin metadata ----

#[no_mangle]
pub extern "C" fn plugin_info() -> u64 {
    let info = serde_json::json!({
        "name": "{{name}}",
        "version": "0.1.0",
        "description": "{{name}} plugin for OSG",
        "hooks": ["build.finished"]
    });
    bytes_to_wasm(info.to_string().into_bytes())
}

// ---- Event handling ----

#[derive(Deserialize)]
struct Event {
    #[serde(rename = "type")]
    event_type: String,
    payload: Value,
}

/// Main entry point called by the OSG host for each hook.
///
/// Available hooks:
///   - config.validate    — validate config (errors abort build)
///   - build.started      — build pipeline starting
///   - content.transform  — modify Markdown before rendering
///   - page.render        — override page template context
///   - section.render     — override section template context
///   - taxonomy.list.render — override taxonomy list context
///   - taxonomy.term.render — override taxonomy term context
///   - image.process      — process images via WASI filesystem
///   - build.finished     — all rendering complete
///   - after.build        — post-build (deploy, notifications)
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
        "build.finished" => on_build_finished(&event.payload),
        _ => 0,
    }
}

// ---- Hook handlers ----

fn on_build_finished(payload: &Value) -> u64 {
    let public_dir = payload
        .get("config")
        .and_then(|c| c.get("public_dir"))
        .and_then(|v| v.as_str())
        .unwrap_or("");

    if public_dir.is_empty() {
        return 0;
    }

    // TODO: implement your plugin logic here.
    // Example: write a file to public_dir via WASI filesystem:
    //   let path = std::path::Path::new(public_dir).join("my-file.txt");
    //   let _ = std::fs::write(&path, "Hello from {{name}}!");

    // Return 0 for no overrides, or return a JSON response to modify
    // the template context:
    //   let resp = serde_json::json!({"payload": {"key": "value"}});
    //   bytes_to_wasm(resp.to_string().into_bytes())
    0
}

// ---- Helpers ----

fn bytes_to_wasm(data: Vec<u8>) -> u64 {
    if data.is_empty() {
        return 0;
    }
    let len = data.len() as u32;
    let ptr = alloc(len as i32) as u32;
    unsafe {
        std::ptr::copy_nonoverlapping(data.as_ptr(), ptr as *mut u8, len as usize);
    }
    ((ptr as u64) << 32) | (len as u64)
}

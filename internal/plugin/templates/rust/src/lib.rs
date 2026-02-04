use serde::Deserialize;
use serde_json::Value;
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

    // TODO: implement your plugin logic here.
    0
}

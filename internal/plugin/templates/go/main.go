//go:build ignore

// {{name}} — OSG WASM plugin (TinyGo)
//
// Build: ./build.sh
// Install: cp {{name}}.wasm <site>/plugins/
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"unsafe"
)

// ---- Plugin metadata ----

type pluginMeta struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Hooks       []string `json:"hooks"`
}

var meta = pluginMeta{
	Name:        "{{name}}",
	Version:     "0.1.0",
	Description: "{{name}} plugin for OSG",
	Hooks:       []string{"build.finished"},
}

// ---- ABI types ----

type event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type response struct {
	Payload map[string]any `json:"payload"`
}

// ---- WASM ABI exports ----

//go:wasmexport alloc
func alloc(size int32) int32 {
	buf := make([]byte, size)
	if len(buf) == 0 {
		return 0
	}
	return int32(uintptr(unsafe.Pointer(&buf[0])))
}

//go:wasmexport dealloc
func dealloc(ptr int32, size int32) {
	// TinyGo GC handles deallocation; this is a no-op.
}

//go:wasmexport plugin_info
func pluginInfo() uint64 {
	data, err := json.Marshal(meta)
	if err != nil {
		return 0
	}
	return bytesToWasm(data)
}

//go:wasmexport handle_event
func handleEvent(ptr int32, size int32) uint64 {
	if ptr == 0 || size == 0 {
		return 0
	}

	data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	var ev event
	if err := json.Unmarshal(data, &ev); err != nil {
		return 0
	}

	switch ev.Type {
	case "build.finished":
		return onBuildFinished(ev.Payload)
	default:
		return 0
	}
}

// ---- Hook handlers ----

func onBuildFinished(payload map[string]any) uint64 {
	// Example: write a file to public_dir via WASI filesystem.
	publicDir := getString(payload, "config", "public_dir")
	if publicDir == "" {
		return 0
	}

	content := "Hello from {{name}} plugin!\n"
	path := filepath.Join(publicDir, "{{name}}.txt")
	_ = os.WriteFile(path, []byte(content), 0o644)

	// Return 0 for no overrides, or return a response to modify
	// the template context:
	//   resp := response{Payload: map[string]any{"key": "value"}}
	//   data, _ := json.Marshal(resp)
	//   return bytesToWasm(data)
	return 0
}

// ---- Helpers ----

func bytesToWasm(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}
	ptr := alloc(int32(len(data)))
	dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(data))
	copy(dest, data)
	return (uint64(uint32(ptr)) << 32) | uint64(uint32(len(data)))
}

func getString(m map[string]any, keys ...string) string {
	current := any(m)
	for _, key := range keys {
		cm, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = cm[key]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

func main() {} // required by TinyGo, but unused

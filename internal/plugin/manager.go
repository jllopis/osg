package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Manager struct {
	runtime wazero.Runtime
	plugins []*Plugin
	logger  *slog.Logger
	timeout time.Duration // per-plugin call timeout; zero means no timeout
}

type Plugin struct {
	name      string
	module    api.Module
	memory    api.Memory
	allocFn   api.Function
	deallocFn api.Function
	handleFn  api.Function
	info      PluginMeta
}

// PluginMeta holds optional metadata exported by a plugin via plugin_info.
type PluginMeta struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Hooks       []string `json:"hooks"`
}

type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type Response struct {
	Payload map[string]any `json:"payload"`
}

// Load discovers and loads enabled .wasm plugins from dir.
// timeoutSec sets the per-call timeout in seconds (0 = no timeout).
func Load(ctx context.Context, dir string, enabled []string, timeoutSec int, logger *slog.Logger) (*Manager, error) {
	var timeout time.Duration
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	if strings.TrimSpace(dir) == "" {
		return &Manager{logger: logger, timeout: timeout}, nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manager{logger: logger, timeout: timeout}, nil
		}
		return nil, fmt.Errorf("stat plugins dir: %w", err)
	}
	if !info.IsDir() {
		return &Manager{logger: logger, timeout: timeout}, nil
	}

	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return nil, fmt.Errorf("init wasi: %w", err)
	}

	manager := &Manager{runtime: runtime, logger: logger, timeout: timeout}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}

	enabledSet := map[string]bool{}
	for _, name := range enabled {
		name = normalizePluginName(name)
		if name == "" {
			continue
		}
		enabledSet[name] = true
	}
	if len(enabledSet) == 0 {
		return manager, nil
	}

	available := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".wasm") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		available[name] = true
		if !enabledSet[name] {
			continue
		}
		plugin, err := loadPlugin(ctx, runtime, path)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to load plugin", "path", path, "error", err)
			}
			continue
		}
		manager.plugins = append(manager.plugins, plugin)
	}

	if logger != nil {
		for name := range enabledSet {
			if !available[name] {
				logger.Warn("plugin enabled but not installed", "plugin", name)
			}
		}
	}

	return manager, nil
}

func normalizePluginName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".wasm") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return strings.TrimSpace(name)
}

// Metadata returns metadata for all loaded plugins.
func (m *Manager) Metadata() []PluginMeta {
	out := make([]PluginMeta, len(m.plugins))
	for i, p := range m.plugins {
		out[i] = p.info
	}
	return out
}

func (m *Manager) Close(ctx context.Context) error {
	if m.runtime == nil {
		return nil
	}
	return m.runtime.Close(ctx)
}

// pluginResult holds the outcome of a single plugin call.
type pluginResult struct {
	name    string
	resp    *Response
	err     error
	elapsed time.Duration
}

func (m *Manager) Emit(ctx context.Context, event string, payload map[string]any) map[string]any {
	if len(m.plugins) == 0 {
		return nil
	}

	request := Event{Type: event, Payload: payload}
	data, err := json.Marshal(request)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("plugin event marshal failed", "event", event, "error", err)
		}
		return nil
	}

	// Single plugin: avoid goroutine overhead.
	if len(m.plugins) == 1 {
		return m.emitSingle(ctx, m.plugins[0], event, data)
	}

	// Multiple plugins: run in parallel, collect results.
	results := make([]pluginResult, len(m.plugins))
	var wg sync.WaitGroup
	wg.Add(len(m.plugins))

	for i, p := range m.plugins {
		go func(idx int, plugin *Plugin) {
			defer wg.Done()
			callCtx := ctx
			var cancel context.CancelFunc
			if m.timeout > 0 {
				callCtx, cancel = context.WithTimeout(ctx, m.timeout)
			}

			start := time.Now()
			resp, err := plugin.Call(callCtx, data)
			elapsed := time.Since(start)

			if cancel != nil {
				cancel()
			}

			results[idx] = pluginResult{
				name:    plugin.name,
				resp:    resp,
				err:     err,
				elapsed: elapsed,
			}
		}(i, p)
	}
	wg.Wait()

	// Merge results in original order (alphabetical by plugin name)
	// to guarantee deterministic output.
	overrides := map[string]any{}
	for _, r := range results {
		if r.err != nil {
			if m.logger != nil {
				m.logger.Warn("plugin call failed", "plugin", r.name, "event", event, "elapsed", r.elapsed, "error", r.err)
			}
			continue
		}
		if m.logger != nil {
			m.logger.Debug("plugin call", "plugin", r.name, "event", event, "elapsed", r.elapsed)
		}
		if r.resp == nil || r.resp.Payload == nil {
			continue
		}
		mergeMaps(overrides, r.resp.Payload)
	}

	if len(overrides) == 0 {
		return nil
	}
	return overrides
}

// emitSingle handles the common single-plugin case without goroutine overhead.
func (m *Manager) emitSingle(ctx context.Context, plugin *Plugin, event string, data []byte) map[string]any {
	callCtx := ctx
	var cancel context.CancelFunc
	if m.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, m.timeout)
	}

	start := time.Now()
	resp, err := plugin.Call(callCtx, data)
	elapsed := time.Since(start)

	if cancel != nil {
		cancel()
	}

	if err != nil {
		if m.logger != nil {
			m.logger.Warn("plugin call failed", "plugin", plugin.name, "event", event, "elapsed", elapsed, "error", err)
		}
		return nil
	}
	if m.logger != nil {
		m.logger.Debug("plugin call", "plugin", plugin.name, "event", event, "elapsed", elapsed)
	}
	if resp == nil || resp.Payload == nil {
		return nil
	}
	return resp.Payload
}

func loadPlugin(ctx context.Context, runtime wazero.Runtime, path string) (*Plugin, error) {
	code, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	compiled, err := runtime.CompileModule(ctx, code)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	// Mount the host filesystem at "/" so plugins can write to public_dir
	// and other paths via WASI file operations (e.g. std::fs::write in Rust).
	fsConfig := wazero.NewFSConfig().WithDirMount("/", "/")
	modConfig := wazero.NewModuleConfig().
		WithName(name).
		WithFSConfig(fsConfig).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	module, err := runtime.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		return nil, err
	}

	memory := module.Memory()
	if memory == nil {
		return nil, fmt.Errorf("plugin %s has no exported memory", name)
	}

	allocFn := module.ExportedFunction("alloc")
	handleFn := module.ExportedFunction("handle_event")
	if handleFn == nil || allocFn == nil {
		return nil, fmt.Errorf("plugin %s missing required exports", name)
	}

	deallocFn := module.ExportedFunction("dealloc")

	p := &Plugin{
		name:      name,
		module:    module,
		memory:    memory,
		allocFn:   allocFn,
		deallocFn: deallocFn,
		handleFn:  handleFn,
	}

	// Attempt to read optional plugin_info export for metadata.
	p.info = readPluginInfo(ctx, p)

	return p, nil
}

// readPluginInfo calls the optional plugin_info export.
// The function must take no args and return an i64 (packed ptr/len).
// Returns zero-value PluginMeta if the export is missing or fails.
func readPluginInfo(ctx context.Context, p *Plugin) PluginMeta {
	infoFn := p.module.ExportedFunction("plugin_info")
	if infoFn == nil {
		return PluginMeta{Name: p.name}
	}

	ret, err := infoFn.Call(ctx)
	if err != nil || len(ret) == 0 {
		return PluginMeta{Name: p.name}
	}

	out := ret[0]
	outPtr := uint32(out >> 32)
	outLen := uint32(out)
	if outLen == 0 {
		return PluginMeta{Name: p.name}
	}

	data, ok := p.memory.Read(outPtr, outLen)
	if !ok {
		return PluginMeta{Name: p.name}
	}

	var meta PluginMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return PluginMeta{Name: p.name}
	}
	if meta.Name == "" {
		meta.Name = p.name
	}
	return meta
}

func (p *Plugin) Call(ctx context.Context, payload []byte) (*Response, error) {
	results, err := p.allocFn.Call(ctx, uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	ptr := uint32(results[0])
	if ok := p.memory.Write(ptr, payload); !ok {
		return nil, fmt.Errorf("plugin %s memory write failed", p.name)
	}

	ret, err := p.handleFn.Call(ctx, uint64(ptr), uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	if p.deallocFn != nil {
		_, _ = p.deallocFn.Call(ctx, uint64(ptr), uint64(len(payload)))
	}
	if len(ret) == 0 {
		return nil, nil
	}

	out := ret[0]
	outPtr := uint32(out >> 32)
	outLen := uint32(out)
	if outLen == 0 {
		return nil, nil
	}

	data, ok := p.memory.Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("plugin %s memory read failed", p.name)
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if p.deallocFn != nil {
		_, _ = p.deallocFn.Call(ctx, uint64(outPtr), uint64(outLen))
	}

	return &resp, nil
}

func Merge(dst map[string]any, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	mergeMaps(dst, src)
}

func mergeMaps(dst map[string]any, src map[string]any) {
	for key, value := range src {
		if existing, ok := dst[key]; ok {
			dstMap, dstOk := existing.(map[string]any)
			srcMap, srcOk := value.(map[string]any)
			if dstOk && srcOk {
				mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[key] = value
	}
}

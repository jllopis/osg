package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Manager struct {
	runtime wazero.Runtime
	plugins []*Plugin
	logger  *slog.Logger
}

type Plugin struct {
	name      string
	module    api.Module
	memory    api.Memory
	allocFn   api.Function
	deallocFn api.Function
	handleFn  api.Function
}

type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

type Response struct {
	Payload map[string]any `json:"payload"`
}

func Load(ctx context.Context, dir string, logger *slog.Logger) (*Manager, error) {
	if strings.TrimSpace(dir) == "" {
		return &Manager{logger: logger}, nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manager{logger: logger}, nil
		}
		return nil, fmt.Errorf("stat plugins dir: %w", err)
	}
	if !info.IsDir() {
		return &Manager{logger: logger}, nil
	}

	runtime := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return nil, fmt.Errorf("init wasi: %w", err)
	}

	manager := &Manager{runtime: runtime, logger: logger}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read plugins dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".wasm") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		plugin, err := loadPlugin(ctx, runtime, path)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to load plugin", "path", path, "error", err)
			}
			continue
		}
		manager.plugins = append(manager.plugins, plugin)
	}

	return manager, nil
}

func (m *Manager) Close(ctx context.Context) error {
	if m.runtime == nil {
		return nil
	}
	return m.runtime.Close(ctx)
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

	overrides := map[string]any{}
	for _, plugin := range m.plugins {
		resp, err := plugin.Call(ctx, data)
		if err != nil {
			if m.logger != nil {
				m.logger.Warn("plugin call failed", "plugin", plugin.name, "event", event, "error", err)
			}
			continue
		}
		if resp == nil {
			continue
		}
		if resp.Payload == nil {
			continue
		}
		mergeMaps(overrides, resp.Payload)
	}

	if len(overrides) == 0 {
		return nil
	}
	return overrides
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
	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(name))
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

	return &Plugin{
		name:      name,
		module:    module,
		memory:    memory,
		allocFn:   allocFn,
		deallocFn: deallocFn,
		handleFn:  handleFn,
	}, nil
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

package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/operations"
)

// opsPollUntilRunning triggers name and polls Snapshot until the named
// operation is in StateRunning, failing the test after ~1s.
func opsPollUntilRunning(t *testing.T, runner *operations.Runner, name string) {
	t.Helper()
	if _, err := runner.Trigger(name, nil); err != nil {
		t.Fatalf("trigger %s: %v", name, err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, snap := range runner.Snapshot() {
			if snap.Definition.Name == name && snap.State == operations.StateRunning {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %s never reached running state", name)
}

// opsDrainRunner polls Snapshot until no operation is Running/Starting/
// Stopping, so that background goroutines (e.g. the one handleOperationRunFlow
// spawns) finish writing to the on-disk store before t.TempDir cleanup tries
// to remove it. Registered as a t.Cleanup so it runs LIFO before the store's
// own Close cleanup (registered earlier in newTestRunner). Avoids the flaky
// "TempDir RemoveAll cleanup: directory not empty" race.
func opsDrainRunner(t *testing.T, runner *operations.Runner) {
	t.Helper()
	t.Cleanup(func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			busy := false
			for _, snap := range runner.Snapshot() {
				switch snap.State {
				case operations.StateRunning, operations.StateStarting, operations.StateStopping:
					busy = true
				}
			}
			if !busy {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

// opsNilRunnerServer builds a server whose Runner is nil, to exercise the
// 503 service-unavailable branches.
func opsNilRunnerServer(t *testing.T) *Server {
	t.Helper()
	srv, err := NewServer(ServerOptions{
		Version: "test",
		Cfg:     newTestConfig(t),
		Logger:  newTestLogger(),
		Runner:  nil,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv
}

func TestHandleOperationRunHTML(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	ts := httptest.NewServer(s.mux)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/operations/build/run", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Fatalf("Location = %q, want /", loc)
	}
}

func TestHandleOperationRunJSON(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/build/run", nil)
	req.Header.Set("Accept", "application/json")
	req.SetPathValue("name", "build")
	s.handleOperationRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		OK   bool   `json:"ok"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if !body.OK || body.Name != "build" {
		t.Fatalf("body = %+v, want ok=true name=build", body)
	}
}

func TestHandleOperationRunMissingName(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations//run", nil)
	// No path value set -> empty name.
	s.handleOperationRun(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleOperationRunNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/build/run", nil)
	req.SetPathValue("name", "build")
	s.handleOperationRun(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleOperationRunUnknownJSON(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/nope/run", nil)
	req.Header.Set("Accept", "application/json")
	req.SetPathValue("name", "nope")
	s.handleOperationRun(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error == "" {
		t.Fatalf("expected non-empty error, got %q", rec.Body.String())
	}
}

func TestHandleOperationStop(t *testing.T) {
	runner := newTestRunner(t)
	s := newTestServer(t, runner)
	opsPollUntilRunning(t, runner, "serve")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/serve/stop", nil)
	req.SetPathValue("name", "serve")
	s.handleOperationStop(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
}

func TestHandleOperationStopJSON(t *testing.T) {
	runner := newTestRunner(t)
	s := newTestServer(t, runner)
	opsPollUntilRunning(t, runner, "serve")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/serve/stop", nil)
	req.Header.Set("Accept", "application/json")
	req.SetPathValue("name", "serve")
	s.handleOperationStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		OK   bool   `json:"ok"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.OK || body.Name != "serve" {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandleOperationStopNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/serve/stop", nil)
	req.SetPathValue("name", "serve")
	s.handleOperationStop(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleOperationRunFlowNotInFlow(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/serve/run-flow", nil)
	req.SetPathValue("name", "serve")
	s.handleOperationRunFlow(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleOperationRunFlowHTML(t *testing.T) {
	runner := newTestRunner(t)
	opsDrainRunner(t, runner)
	s := newTestServer(t, runner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/build/run-flow", nil)
	req.SetPathValue("name", "build")
	s.handleOperationRunFlow(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/actions" {
		t.Fatalf("Location = %q, want /actions", loc)
	}
}

func TestHandleOperationRunFlowJSON(t *testing.T) {
	runner := newTestRunner(t)
	opsDrainRunner(t, runner)
	s := newTestServer(t, runner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/build/run-flow", nil)
	req.Header.Set("Accept", "application/json")
	req.SetPathValue("name", "build")
	s.handleOperationRunFlow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		OK    bool     `json:"ok"`
		Chain []string `json:"chain"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.OK || len(body.Chain) == 0 || body.Chain[0] != "build" {
		t.Fatalf("body = %+v, want chain starting at build", body)
	}
}

func TestHandleOperationRunFlowNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operations/build/run-flow", nil)
	req.SetPathValue("name", "build")
	s.handleOperationRunFlow(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleOperationCardStyles(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	styles := []string{"flow-node", "op-card", "quick-button", "task-form"}
	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			// Regression guard: style "op-card" must map to the
			// operation-card.html partial (filename != style id). A
			// mismatch makes ExecuteTemplate fail silently and return an
			// empty 200 body, blanking in-place card refreshes on /actions.
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/operations/build/card?style="+style, nil)
			req.SetPathValue("name", "build")
			s.handleOperationCard(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
			}
			if strings.TrimSpace(rec.Body.String()) == "" {
				t.Fatalf("empty body for style %s", style)
			}
		})
	}
}

func TestHandleOperationCardUnknownStyle(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operations/build/card?style=bogus", nil)
	req.SetPathValue("name", "build")
	s.handleOperationCard(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleOperationCardUnknownOp(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operations/ghost/card?style=op-card", nil)
	req.SetPathValue("name", "ghost")
	s.handleOperationCard(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleOperationsJSON(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operations.json", nil)
	s.handleOperationsJSON(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var out struct {
		Now        string `json:"now"`
		Operations []struct {
			Name string `json:"name"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := map[string]bool{}
	for _, op := range out.Operations {
		names[op.Name] = true
	}
	for _, want := range []string{"build", "deploy", "check", "serve"} {
		if !names[want] {
			t.Fatalf("missing operation %q in %v", want, names)
		}
	}
}

func TestHandleOperationsJSONNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operations.json", nil)
	s.handleOperationsJSON(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleServicesJSON(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/services.json", nil)
	s.handleServicesJSON(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Services []struct {
			Name string `json:"name"`
		} `json:"services"`
		Now string `json:"now"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, svc := range out.Services {
		if svc.Name == "serve" {
			found = true
		}
	}
	if !found {
		t.Fatalf("serve service missing from %v", out.Services)
	}
}

func TestHandleServicesJSONNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/services.json", nil)
	s.handleServicesJSON(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleRebuild(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rebuild", nil)
	s.handleRebuild(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/assets" {
		t.Fatalf("Location = %q, want /assets", loc)
	}
}

func TestHandleRebuildNilRunner(t *testing.T) {
	s := opsNilRunnerServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rebuild", nil)
	s.handleRebuild(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHandleRebuildJSON(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rebuild.json", nil)
	s.handleRebuildJSON(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := out["available"]; !ok {
		t.Fatalf("missing available key in %v", out)
	}
	if _, ok := out["running"]; !ok {
		t.Fatalf("missing running key in %v", out)
	}
}

func TestHandleServiceStartMethodNotAllowed(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/services/start", nil)
	s.handleServiceStart(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleServiceStartMissingName(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/start", nil)
	s.handleServiceStart(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleServiceStartHappyPath(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	form := url.Values{"name": {"serve"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.handleServiceStart(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/services" {
		t.Fatalf("Location = %q, want /services", loc)
	}
}

func TestHandleServiceStopMethodNotAllowed(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/services/stop", nil)
	s.handleServiceStop(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleServiceStopMissingName(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/stop", nil)
	s.handleServiceStop(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleServiceStopHappyPath(t *testing.T) {
	runner := newTestRunner(t)
	s := newTestServer(t, runner)
	opsPollUntilRunning(t, runner, "serve")

	form := url.Values{"name": {"serve"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/stop", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.handleServiceStop(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
}

func TestHandlePluginToggleMissingName(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/toggle", nil)
	s.handlePluginToggle(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePluginToggleAddsAndRemoves(t *testing.T) {
	runner := newTestRunner(t)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.UpdatePluginsEnabled(cfgPath, nil); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	srv, err := NewServer(ServerOptions{
		Version:    "test",
		Cfg:        newTestConfig(t),
		ConfigPath: cfgPath,
		Logger:     newTestLogger(),
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	toggle := func() *httptest.ResponseRecorder {
		form := url.Values{"name": {"my-plugin"}}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/plugins/toggle", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		srv.handlePluginToggle(rec, req)
		return rec
	}

	// First toggle: enable -> name should appear in file.
	rec := toggle()
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("enable status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/plugins" {
		t.Fatalf("Location = %q, want /plugins", loc)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "my-plugin") {
		t.Fatalf("config does not list my-plugin after enable:\n%s", data)
	}

	// Second toggle: disable -> name removed from file.
	rec = toggle()
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("disable status = %d, want 303", rec.Code)
	}
	data, err = os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "my-plugin") {
		t.Fatalf("config still lists my-plugin after disable:\n%s", data)
	}
}

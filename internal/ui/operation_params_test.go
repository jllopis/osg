package ui

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// operationParamsAcceptRequest builds a GET request carrying the given
// Accept header. File-unique helper name to avoid collisions with the
// shared testsupport_test.go helpers.
func operationParamsAcceptRequest(t *testing.T, accept string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	return req
}

func TestFlowDownstream(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "start of flow returns full chain",
			in:   "init",
			want: []string{"init", "update-content", "check", "build", "deploy"},
		},
		{
			name: "middle returns suffix",
			in:   "check",
			want: []string{"check", "build", "deploy"},
		},
		{
			name: "last returns single",
			in:   "deploy",
			want: []string{"deploy"},
		},
		{
			name: "unknown returns nil",
			in:   "does-not-exist",
			want: nil,
		},
		{
			name: "empty returns nil",
			in:   "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flowDownstream(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("flowDownstream(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFlowDownstreamReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not corrupt the package-level
	// actionFlow backing array.
	got := flowDownstream("init")
	if len(got) == 0 {
		t.Fatal("expected non-empty chain for init")
	}
	got[0] = "MUTATED"
	if actionFlow[0] == "MUTATED" {
		t.Fatal("flowDownstream returned a slice aliasing actionFlow; mutation leaked")
	}
}

func TestPageForOperation(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"init", "actions"},
		{"build", "actions"},
		{"deploy", "actions"},
		{"update-content", "actions"},
		{"check", "actions"},
		{"audit", "audit"},
		{"new", "vault"},
		{"theme-init", "themes"},
		{"plugin-install", "plugins"},
		{"import-wordpress", "import"},
		{"import-hugo", "import"},
		{"serve", "services"},
		{"api", "services"},
		{"watcher", "services"},
		{"scheduler", "services"},
		{"totally-unknown", "actions"},
		{"", "actions"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := pageForOperation(tt.in); got != tt.want {
				t.Fatalf("pageForOperation(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOperationsForPage(t *testing.T) {
	views := []OperationView{
		{Name: "init"},
		{Name: "audit"},
		{Name: "build"},
		{Name: "new"},
		{Name: "deploy"},
		{Name: "theme-init"},
		{Name: "unknown-op"}, // defaults to "actions"
	}

	t.Run("filters actions and preserves order", func(t *testing.T) {
		got := operationsForPage(views, "actions")
		var names []string
		for _, v := range got {
			names = append(names, v.Name)
		}
		want := []string{"init", "build", "deploy", "unknown-op"}
		if !reflect.DeepEqual(names, want) {
			t.Fatalf("operationsForPage(actions) names = %v, want %v", names, want)
		}
	})

	t.Run("filters single-member page", func(t *testing.T) {
		got := operationsForPage(views, "vault")
		if len(got) != 1 || got[0].Name != "new" {
			t.Fatalf("operationsForPage(vault) = %v, want single [new]", got)
		}
	})

	t.Run("no matches returns empty non-nil", func(t *testing.T) {
		got := operationsForPage(views, "nonexistent-page")
		if got == nil {
			t.Fatal("operationsForPage should return non-nil slice")
		}
		if len(got) != 0 {
			t.Fatalf("operationsForPage(nonexistent) = %v, want empty", got)
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		got := operationsForPage(nil, "actions")
		if len(got) != 0 {
			t.Fatalf("operationsForPage(nil) = %v, want empty", got)
		}
	})
}

func TestHasCollapsibleParams(t *testing.T) {
	if !hasCollapsibleParams("check") {
		t.Fatal("expected check to have collapsible params")
	}
	if hasCollapsibleParams("build") {
		t.Fatal("expected build NOT to have collapsible params")
	}
	if hasCollapsibleParams("unknown") {
		t.Fatal("expected unknown op NOT to have collapsible params")
	}
}

func TestParamsForOperation(t *testing.T) {
	t.Run("unknown returns nil", func(t *testing.T) {
		if got := paramsForOperation("no-such-op"); got != nil {
			t.Fatalf("paramsForOperation(unknown) = %v, want nil", got)
		}
	})

	cases := []struct {
		op         string
		wantFields []string
	}{
		{"build", []string{"force-ai-summaries"}},
		{"deploy", []string{"provider", "preview", "build"}},
		{"check", []string{"links", "images", "frontmatter", "json"}},
		{"audit", []string{"json"}},
		{"new", []string{"title", "tags", "publish", "notes-dir"}},
		{"theme-init", []string{"name", "parent"}},
		{"plugin-install", []string{"path"}},
		{"import-wordpress", []string{"file"}},
		{"import-hugo", []string{"dir"}},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			got := paramsForOperation(tc.op)
			if got == nil {
				t.Fatalf("paramsForOperation(%q) = nil, want non-nil schema", tc.op)
			}
			var names []string
			for _, p := range got {
				names = append(names, p.Name)
			}
			if !reflect.DeepEqual(names, tc.wantFields) {
				t.Fatalf("paramsForOperation(%q) field names = %v, want %v", tc.op, names, tc.wantFields)
			}
		})
	}
}

func TestConfirmTextFor(t *testing.T) {
	confirms := []string{"deploy", "import-wordpress", "import-hugo"}
	for _, op := range confirms {
		if confirmTextFor(op) == "" {
			t.Fatalf("confirmTextFor(%q) = empty, want non-empty confirmation text", op)
		}
	}
	noConfirm := []string{"build", "check", "new", "audit", "unknown-op", ""}
	for _, op := range noConfirm {
		if got := confirmTextFor(op); got != "" {
			t.Fatalf("confirmTextFor(%q) = %q, want empty", op, got)
		}
	}
}

// TestRegistryInvariants asserts the cross-map invariants that keep the
// dashboard wiring consistent.
func TestRegistryInvariants(t *testing.T) {
	t.Run("every actionFlow name lives on the actions page", func(t *testing.T) {
		for _, name := range actionFlow {
			if page := pageForOperation(name); page != "actions" {
				t.Errorf("actionFlow op %q is on page %q, want actions", name, page)
			}
		}
	})

	t.Run("every confirmOperations key exists in the param registry", func(t *testing.T) {
		for name := range confirmOperations {
			if _, ok := operationParamRegistry[name]; !ok {
				t.Errorf("confirmOperations key %q has no entry in operationParamRegistry", name)
			}
		}
	})

	t.Run("required string params are kind string", func(t *testing.T) {
		for op, defs := range operationParamRegistry {
			for _, d := range defs {
				if d.Required && d.Kind != "string" {
					t.Errorf("op %q field %q is required but Kind=%q, want string", op, d.Name, d.Kind)
				}
			}
		}
	})
}

func TestParamsFromForm(t *testing.T) {
	t.Run("empty form returns nil", func(t *testing.T) {
		if got := paramsFromForm(map[string][]string{}); got != nil {
			t.Fatalf("paramsFromForm(empty) = %v, want nil", got)
		}
		if got := paramsFromForm(nil); got != nil {
			t.Fatalf("paramsFromForm(nil) = %v, want nil", got)
		}
	})

	t.Run("only empty values returns nil", func(t *testing.T) {
		form := map[string][]string{
			"a": {""},
			"b": {},
		}
		if got := paramsFromForm(form); got != nil {
			t.Fatalf("paramsFromForm(all-empty) = %v, want nil", got)
		}
	})

	t.Run("coercion", func(t *testing.T) {
		form := map[string][]string{
			"flagTrue":  {"true"},
			"flagFalse": {"false"},
			"count":     {"42"},
			"negative":  {"-7"},
			"text":      {"hello"},
			"empty":     {""},
			"none":      {},
			"multi":     {"first", "second"}, // only first is read
		}
		got := paramsFromForm(form)

		if v, ok := got["flagTrue"].(bool); !ok || v != true {
			t.Errorf("flagTrue = %#v, want bool true", got["flagTrue"])
		}
		if v, ok := got["flagFalse"].(bool); !ok || v != false {
			t.Errorf("flagFalse = %#v, want bool false", got["flagFalse"])
		}
		if v, ok := got["count"].(int64); !ok || v != 42 {
			t.Errorf("count = %#v, want int64 42", got["count"])
		}
		if v, ok := got["negative"].(int64); !ok || v != -7 {
			t.Errorf("negative = %#v, want int64 -7", got["negative"])
		}
		if v, ok := got["text"].(string); !ok || v != "hello" {
			t.Errorf("text = %#v, want string hello", got["text"])
		}
		if _, present := got["empty"]; present {
			t.Error("empty value should be dropped")
		}
		if _, present := got["none"]; present {
			t.Error("zero-length value slice should be dropped")
		}
		if v, ok := got["multi"].(string); !ok || v != "first" {
			t.Errorf("multi = %#v, want string first", got["multi"])
		}
	})
}

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{"plain json", "application/json", true},
		{"json with q value", "application/json;q=0.9", true},
		{"json among many", "text/html, application/json, */*", true},
		{"json with q among many", "text/html;q=0.8, application/json;q=0.9", true},
		{"html only", "text/html", false},
		{"empty header", "", false},
		{"wildcard only", "*/*", false},
		{"trailing whitespace", " application/json ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := operationParamsAcceptRequest(t, tt.accept)
			if got := wantsJSON(req); got != tt.want {
				t.Fatalf("wantsJSON(Accept=%q) = %v, want %v", tt.accept, got, tt.want)
			}
		})
	}
}

func TestSplitAcceptHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single", "application/json", []string{"application/json"}},
		{"comma separated", "text/html, application/json", []string{"text/html", "application/json"}},
		// splitAcceptHeader splits on both ',' and ';' and keeps every
		// non-empty trimmed segment; it does NOT discard q-values or
		// media-type parameters. wantsJSON tolerates the extra segments
		// because it only exact-matches "application/json".
		{"q values become separate segments", "text/html;q=0.8, application/json;q=0.9", []string{"text/html", "q=0.8", "application/json", "q=0.9"}},
		{"whitespace trimmed", "  text/html  ,  application/json  ", []string{"text/html", "application/json"}},
		{"semicolon params become separate segments", "application/json;charset=utf-8", []string{"application/json", "charset=utf-8"}},
		{"empty segments dropped", "text/html,,application/json", []string{"text/html", "application/json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAcceptHeader(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitAcceptHeader(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestTrimSpaceASCII(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc", "abc"},
		{"  abc", "abc"},
		{"abc  ", "abc"},
		{"  abc  ", "abc"},
		{"\tabc\t", "abc"},
		{" \t abc \t ", "abc"},
		{"   ", ""},
		{"a b c", "a b c"}, // interior whitespace preserved
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := trimSpaceASCII(tt.in); got != tt.want {
				t.Fatalf("trimSpaceASCII(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

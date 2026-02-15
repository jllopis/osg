package publish

import "testing"

func TestShouldPublish(t *testing.T) {
	cases := []struct {
		name   string
		fm     map[string]any
		expect bool
		draft  bool
	}{
		{
			name:   "bool true",
			fm:     map[string]any{"publish": true},
			expect: true,
			draft:  false,
		},
		{
			name:   "string true",
			fm:     map[string]any{"publish": "true"},
			expect: true,
			draft:  false,
		},
		{
			name:   "string draft",
			fm:     map[string]any{"publish": "draft"},
			expect: true,
			draft:  true,
		},
		{
			name:   "missing",
			fm:     map[string]any{},
			expect: false,
			draft:  false,
		},
		// osg block tests
		{
			name:   "osg.publish bool true",
			fm:     map[string]any{"osg": map[string]any{"publish": true}},
			expect: true,
			draft:  false,
		},
		{
			name:   "osg.publish string draft",
			fm:     map[string]any{"osg": map[string]any{"publish": "draft"}},
			expect: true,
			draft:  true,
		},
		{
			name:   "osg.publish false",
			fm:     map[string]any{"osg": map[string]any{"publish": false}},
			expect: false,
			draft:  false,
		},
		{
			name:   "osg.publish overrides top-level publish",
			fm:     map[string]any{"publish": true, "osg": map[string]any{"publish": false}},
			expect: false,
			draft:  false,
		},
		{
			name:   "osg block without publish falls back to top-level",
			fm:     map[string]any{"publish": true, "osg": map[string]any{"featured": true}},
			expect: true,
			draft:  false,
		},
		{
			name:   "osg block wrong type ignored",
			fm:     map[string]any{"publish": true, "osg": "not a map"},
			expect: true,
			draft:  false,
		},
	}

	for _, tc := range cases {
		publish, draft := ShouldPublish(tc.fm)
		if publish != tc.expect || draft != tc.draft {
			t.Fatalf("%s: expected publish=%v draft=%v, got publish=%v draft=%v", tc.name, tc.expect, tc.draft, publish, draft)
		}
	}
}

func TestGetOSGString(t *testing.T) {
	cases := []struct {
		name   string
		fm     map[string]any
		key    string
		expect string
	}{
		{
			name:   "existing string key",
			fm:     map[string]any{"osg": map[string]any{"path": "about"}},
			key:    "path",
			expect: "about",
		},
		{
			name:   "string with whitespace",
			fm:     map[string]any{"osg": map[string]any{"path": "  about  "}},
			key:    "path",
			expect: "about",
		},
		{
			name:   "missing key",
			fm:     map[string]any{"osg": map[string]any{"publish": true}},
			key:    "path",
			expect: "",
		},
		{
			name:   "no osg block",
			fm:     map[string]any{"title": "Test"},
			key:    "path",
			expect: "",
		},
		{
			name:   "nil frontmatter",
			fm:     nil,
			key:    "path",
			expect: "",
		},
		{
			name:   "non-string value",
			fm:     map[string]any{"osg": map[string]any{"path": 42}},
			key:    "path",
			expect: "",
		},
		{
			name:   "osg block wrong type",
			fm:     map[string]any{"osg": "not a map"},
			key:    "path",
			expect: "",
		},
	}

	for _, tc := range cases {
		got := GetOSGString(tc.fm, tc.key)
		if got != tc.expect {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.expect, got)
		}
	}
}

func TestGetOSGBool(t *testing.T) {
	cases := []struct {
		name   string
		fm     map[string]any
		key    string
		expect bool
	}{
		{
			name:   "bool true",
			fm:     map[string]any{"osg": map[string]any{"menu": true}},
			key:    "menu",
			expect: true,
		},
		{
			name:   "bool false",
			fm:     map[string]any{"osg": map[string]any{"menu": false}},
			key:    "menu",
			expect: false,
		},
		{
			name:   "string true",
			fm:     map[string]any{"osg": map[string]any{"menu": "true"}},
			key:    "menu",
			expect: true,
		},
		{
			name:   "string TRUE case insensitive",
			fm:     map[string]any{"osg": map[string]any{"menu": "TRUE"}},
			key:    "menu",
			expect: true,
		},
		{
			name:   "string false",
			fm:     map[string]any{"osg": map[string]any{"menu": "false"}},
			key:    "menu",
			expect: false,
		},
		{
			name:   "missing key",
			fm:     map[string]any{"osg": map[string]any{"publish": true}},
			key:    "menu",
			expect: false,
		},
		{
			name:   "no osg block",
			fm:     map[string]any{"title": "Test"},
			key:    "menu",
			expect: false,
		},
		{
			name:   "nil frontmatter",
			fm:     nil,
			key:    "menu",
			expect: false,
		},
		{
			name:   "non-bool value",
			fm:     map[string]any{"osg": map[string]any{"menu": 42}},
			key:    "menu",
			expect: false,
		},
	}

	for _, tc := range cases {
		got := GetOSGBool(tc.fm, tc.key)
		if got != tc.expect {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.expect, got)
		}
	}
}

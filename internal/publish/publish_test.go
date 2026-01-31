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
	}

	for _, tc := range cases {
		publish, draft := ShouldPublish(tc.fm)
		if publish != tc.expect || draft != tc.draft {
			t.Fatalf("%s: expected publish=%v draft=%v, got publish=%v draft=%v", tc.name, tc.expect, tc.draft, publish, draft)
		}
	}
}

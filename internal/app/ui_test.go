package app

import "testing"

func TestNormalizeLoopbackAddr(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{":1314", "127.0.0.1:1314", false},
		{"127.0.0.1:1314", "127.0.0.1:1314", false},
		{"localhost:8080", "localhost:8080", false},
		{"[::1]:1314", "[::1]:1314", false},
		{"0.0.0.0:1314", "", true},
		{"192.168.1.5:1314", "", true},
		{"example.com:80", "", true},
		{"badaddr", "", true},
	}
	for _, c := range cases {
		got, err := normalizeLoopbackAddr(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeLoopbackAddr(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeLoopbackAddr(%q) err=%v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeLoopbackAddr(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

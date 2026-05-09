package app

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"osg/internal/operations"
)

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

func TestAutostartServices_StartsListedServices(t *testing.T) {
	var startedSvc, startedTask atomic.Int32
	defs := []operations.Definition{
		{Name: "test-service", Kind: operations.KindService, Run: func(ctx context.Context, _ map[string]any, _ io.Writer) error {
			startedSvc.Add(1)
			<-ctx.Done()
			return nil
		}},
		{Name: "test-task", Kind: operations.KindTask, Run: func(ctx context.Context, _ map[string]any, _ io.Writer) error {
			startedTask.Add(1)
			return nil
		}},
	}
	runner := operations.New(defs, nil)
	defer runner.StopAll()

	autostartServices(runner, []string{"test-service", "test-task", "unknown"}, nil)

	// Give the service goroutine a moment to enter Run().
	time.Sleep(80 * time.Millisecond)
	if got := startedSvc.Load(); got != 1 {
		t.Errorf("service started %d times, want 1", got)
	}
	if got := startedTask.Load(); got != 0 {
		t.Errorf("task should be skipped (kind=task is for one-shots), got %d starts", got)
	}
}

func TestAutostartServices_NilRunnerNoOp(t *testing.T) {
	// Must not panic with a nil runner or empty list.
	autostartServices(nil, []string{"scheduler"}, nil)
	autostartServices(operations.New(nil, nil), nil, nil)
}

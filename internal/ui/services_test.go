package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRingBufferWriteAndTail(t *testing.T) {
	r := newRingBuffer(3)
	for i := 0; i < 5; i++ {
		_, _ = fmt.Fprintf(r, "line-%d\n", i)
	}
	got := r.Tail(10)
	want := []string{"line-2", "line-3", "line-4"}
	if !equalSlices(got, want) {
		t.Fatalf("Tail()=%v want %v", got, want)
	}
	got = r.Tail(2)
	if !equalSlices(got, []string{"line-3", "line-4"}) {
		t.Fatalf("Tail(2)=%v", got)
	}
}

func TestRingBufferSubscribeReceivesLive(t *testing.T) {
	r := newRingBuffer(10)
	_, _ = r.Write([]byte("history-1\n"))
	_, _ = r.Write([]byte("history-2\n"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := r.Subscribe(ctx)

	got := drain(t, ch, 2, 200*time.Millisecond)
	if !equalSlices(got, []string{"history-1", "history-2"}) {
		t.Fatalf("history replay=%v", got)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = r.Write([]byte("live-1\n"))
		_, _ = r.Write([]byte("live-2\n"))
	}()
	live := drain(t, ch, 2, 500*time.Millisecond)
	if !equalSlices(live, []string{"live-1", "live-2"}) {
		t.Fatalf("live=%v", live)
	}
}

func TestSupervisorStartStop(t *testing.T) {
	stopCalled := make(chan struct{})
	sup := NewSupervisor([]ServiceMeta{{
		Name: "echo",
		Addr: ":0",
		Runner: func(ctx context.Context, w io.Writer) error {
			_, _ = w.Write([]byte("starting echo\n"))
			<-ctx.Done()
			close(stopCalled)
			return nil
		},
	}})

	if err := sup.Start("echo"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Service should be running after the startup window.
	snap := sup.Snapshot()
	if len(snap) != 1 || snap[0].State != StateRunning {
		t.Fatalf("expected running, got %+v", snap)
	}

	if err := sup.Start("echo"); err == nil {
		t.Fatalf("expected error starting already-running service")
	}

	if err := sup.Stop("echo"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case <-stopCalled:
	case <-time.After(time.Second):
		t.Fatalf("runner did not observe ctx cancellation")
	}

	snap = sup.Snapshot()
	if snap[0].State != StateIdle {
		t.Fatalf("expected idle after stop, got %s", snap[0].State)
	}
}

func TestSupervisorStartImmediateError(t *testing.T) {
	sup := NewSupervisor([]ServiceMeta{{
		Name: "broken",
		Runner: func(ctx context.Context, w io.Writer) error {
			return errors.New("port in use")
		},
	}})

	err := sup.Start("broken")
	if err == nil || !strings.Contains(err.Error(), "port in use") {
		t.Fatalf("expected immediate error from runner, got %v", err)
	}

	snap := sup.Snapshot()
	if snap[0].State != StateError {
		t.Fatalf("expected error state, got %s", snap[0].State)
	}
	if snap[0].LastError == "" {
		t.Fatalf("expected LastError to be set")
	}
}

func TestSupervisorUnknownService(t *testing.T) {
	sup := NewSupervisor(nil)
	if err := sup.Start("missing"); err == nil {
		t.Fatalf("expected error starting unknown service")
	}
	if err := sup.Stop("missing"); err == nil {
		t.Fatalf("expected error stopping unknown service")
	}
}

func TestSupervisorStopAllStopsRunning(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	runner := func(ctx context.Context, _ io.Writer) error {
		defer wg.Done()
		<-ctx.Done()
		return nil
	}
	sup := NewSupervisor([]ServiceMeta{
		{Name: "a", Runner: runner},
		{Name: "b", Runner: runner},
	})
	if err := sup.Start("a"); err != nil {
		t.Fatalf("start a: %v", err)
	}
	if err := sup.Start("b"); err != nil {
		t.Fatalf("start b: %v", err)
	}

	sup.StopAll()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("StopAll did not stop all runners")
	}
}

func drain(t *testing.T, ch <-chan string, n int, timeout time.Duration) []string {
	t.Helper()
	out := make([]string, 0, n)
	deadline := time.After(timeout)
	for len(out) < n {
		select {
		case line, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, line)
		case <-deadline:
			return out
		}
	}
	return out
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

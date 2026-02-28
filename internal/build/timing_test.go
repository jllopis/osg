package build

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestBuildTimings_Stage(t *testing.T) {
	bt := &BuildTimings{}

	done := bt.stage("test-stage")
	time.Sleep(5 * time.Millisecond)
	done()

	if len(bt.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(bt.Stages))
	}
	if bt.Stages[0].Name != "test-stage" {
		t.Fatalf("expected stage name 'test-stage', got %q", bt.Stages[0].Name)
	}
	if bt.Stages[0].Duration < 5*time.Millisecond {
		t.Fatalf("expected duration >= 5ms, got %v", bt.Stages[0].Duration)
	}
}

func TestBuildTimings_MultipleStages(t *testing.T) {
	bt := &BuildTimings{}

	done := bt.stage("first")
	done()
	done = bt.stage("second")
	done()
	done = bt.stage("third")
	done()

	if len(bt.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(bt.Stages))
	}

	names := []string{"first", "second", "third"}
	for i, want := range names {
		if bt.Stages[i].Name != want {
			t.Errorf("stage %d: expected %q, got %q", i, want, bt.Stages[i].Name)
		}
	}
}

func TestBuildTimings_Log(t *testing.T) {
	bt := &BuildTimings{}
	done := bt.stage("assets")
	done()
	bt.Total = 100 * time.Millisecond

	// Should not panic with a real logger.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bt.Log(logger)
}

func TestBuildTimings_Log_Empty(t *testing.T) {
	bt := &BuildTimings{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Should not panic with no stages.
	bt.Log(logger)
}

package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	ctx := context.Background()
	opts := Opts{
		Index:   1,
		Command: "echo",
		Args:    []string{"hello"},
	}
	r := Run(ctx, opts)
	if !r.OK() {
		t.Errorf("expected OK, got exit=%d timed_out=%v interrupted=%v", r.ExitCode, r.TimedOut, r.Interrupted)
	}
	if !strings.Contains(r.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", r.Stdout)
	}
	if r.DurationMs == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestRunFailure(t *testing.T) {
	ctx := context.Background()
	opts := Opts{
		Index:   1,
		Command: "sh",
		Args:    []string{"-c", "exit 42"},
	}
	r := Run(ctx, opts)
	if r.OK() {
		t.Error("expected not OK")
	}
	if r.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", r.ExitCode)
	}
	if r.ExitStatus() != 42 {
		t.Errorf("expected exit status 42, got %d", r.ExitStatus())
	}
}

func TestRunTimeout(t *testing.T) {
	ctx := context.Background()
	opts := Opts{
		Index:   1,
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: 100 * time.Millisecond,
	}
	r := Run(ctx, opts)
	if r.OK() {
		t.Error("expected not OK after timeout")
	}
	if !r.TimedOut {
		t.Error("expected TimedOut=true")
	}
	if r.ExitStatus() != 124 {
		t.Errorf("expected exit status 124, got %d", r.ExitStatus())
	}
	if r.Duration < 100*time.Millisecond {
		t.Errorf("expected duration >= 100ms, got %v", r.Duration)
	}
}

func TestRunContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	opts := Opts{
		Index:   1,
		Command: "sleep",
		Args:    []string{"10"},
	}
	r := Run(ctx, opts)
	if r.OK() {
		t.Error("expected not OK after cancel")
	}
	if !r.Interrupted {
		t.Error("expected Interrupted=true")
	}
	if r.ExitStatus() != 130 {
		t.Errorf("expected exit status 130, got %d", r.ExitStatus())
	}
}

func TestRunOutputCapture(t *testing.T) {
	ctx := context.Background()
	var buf strings.Builder
	opts := Opts{
		Index:   1,
		Command: "echo",
		Args:    []string{"line1", "line2"},
		LogW:    &buf,
		LogWErr: &buf,
	}
	r := Run(ctx, opts)
	if !r.OK() {
		t.Error("expected OK")
	}
	output := buf.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Errorf("expected output to contain both lines, got %q", output)
	}
}

func TestRunOnOutput(t *testing.T) {
	ctx := context.Background()
	var lines []string
	opts := Opts{
		Index:   1,
		Command: "echo",
		Args:    []string{"hello"},
		OnOutput: func(line string) {
			lines = append(lines, line)
		},
	}
	r := Run(ctx, opts)
	if !r.OK() {
		t.Error("expected OK")
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [hello], got %v", lines)
	}
}

func TestRunStdoutCapped(t *testing.T) {
	ctx := context.Background()
	opts := Opts{
		Index:   1,
		Command: "sh",
		Args:    []string{"-c", "yes | head -c 70000"},
	}
	r := Run(ctx, opts)
	if len(r.Stdout) > maxBufSize {
		t.Errorf("stdout should be capped at %d, got %d", maxBufSize, len(r.Stdout))
	}
}

func TestRunIndexAndCommand(t *testing.T) {
	ctx := context.Background()
	opts := Opts{
		Index:   5,
		Command: "echo",
		Args:    []string{"x", "y"},
	}
	r := Run(ctx, opts)
	if r.RunIndex != 5 {
		t.Errorf("expected RunIndex 5, got %d", r.RunIndex)
	}
	if !strings.Contains(r.Command, "echo") || !strings.Contains(r.Command, "x y") {
		t.Errorf("expected command to contain 'echo x y', got %q", r.Command)
	}
}

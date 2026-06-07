package summary

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Mephistophiles/repeat/internal/runner"
)

func TestBuild(t *testing.T) {
	results := []runner.Result{
		{RunIndex: 1, ExitCode: 0, DurationMs: 10, Stdout: "ok\n"},
		{RunIndex: 2, ExitCode: 1, DurationMs: 20, Stderr: "fail\n"},
	}

	out := Build("echo test", results)

	if out.Command != "echo test" {
		t.Errorf("expected command 'echo test', got %q", out.Command)
	}
	if out.Total != 2 {
		t.Errorf("expected total 2, got %d", out.Total)
	}
	if out.Successes != 1 {
		t.Errorf("expected 1 success, got %d", out.Successes)
	}
	if out.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", out.Failures)
	}
	if out.TotalDurationMs != 30 {
		t.Errorf("expected total 30ms, got %d", out.TotalDurationMs)
	}
}

func TestWrite(t *testing.T) {
	results := []runner.Result{
		{RunIndex: 1, ExitCode: 0, DurationMs: 5, Stdout: "hello\n"},
	}
	out := Build("echo", results)

	var buf bytes.Buffer
	if err := Write(&buf, out); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var decoded Output
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Total != 1 || decoded.Successes != 1 {
		t.Errorf("unexpected decoded: %+v", decoded)
	}
}

func TestBuildEmpty(t *testing.T) {
	out := Build("cmd", nil)
	if out.Total != 0 {
		t.Errorf("expected total 0, got %d", out.Total)
	}
	if out.Successes != 0 {
		t.Errorf("expected 0 successes, got %d", out.Successes)
	}
}

func TestBuildAllFailures(t *testing.T) {
	results := []runner.Result{
		{RunIndex: 1, ExitCode: 1},
		{RunIndex: 2, TimedOut: true},
		{RunIndex: 3, Interrupted: true},
	}
	out := Build("failing", results)

	if out.Successes != 0 {
		t.Errorf("expected 0 successes, got %d", out.Successes)
	}
	if out.Failures != 3 {
		t.Errorf("expected 3 failures, got %d", out.Failures)
	}
}

func TestWriteIndent(t *testing.T) {
	out := Build("cmd", nil)
	var buf bytes.Buffer
	Write(&buf, out)
	if !strings.Contains(buf.String(), "  ") {
		t.Error("expected indented JSON")
	}
}

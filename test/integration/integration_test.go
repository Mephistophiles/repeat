//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "repeat")
	moduleRoot := filepath.Join("..", "..")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/repeat")
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestBasicRun(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "3", "echo", "hello")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repeat: %v\n%s", err, out)
	}

	if n := countRunLogs(dir); n != 3 {
		t.Errorf("expected 3 log files, got %d", n)
	}
}

func TestErrorExitCode(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "2", "sh", "-c", "exit 5")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 5 {
			t.Errorf("expected exit code 5, got %d", exitErr.ExitCode())
		}
	} else {
		t.Fatalf("expected ExitError, got %T", err)
	}
}

func TestStopOnFirstError(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "5", "sh", "-c", "exit 1")
	cmd.Dir = dir
	cmd.Run()

	if n := countRunLogs(dir); n != 1 {
		t.Errorf("expected only 1 log file, got %d", n)
	}
}

func TestContinue(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "--continue", "3", "sh", "-c", "exit 1")
	cmd.Dir = dir
	cmd.Run()

	if n := countRunLogs(dir); n != 3 {
		t.Errorf("expected 3 log files, got %d", n)
	}
}

func TestVerbose(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "-v", "1", "echo", "hello")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repeat: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("expected 'hello' in output, got %q", string(out))
	}
}

func TestJSON(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "--json", "2", "echo", "hello")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("repeat: %v\n%s", err, out)
	}

	var result struct {
		Total     int `json:"total"`
		Successes int `json:"successes"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, string(out))
	}
	if result.Total != 2 || result.Successes != 2 {
		t.Errorf("expected total=2 successes=2, got %+v", result)
	}
}

func TestTimeout(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "-t", "200ms", "1", "sleep", "10")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for timeout")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 124 {
			t.Errorf("expected exit code 124, got %d", exitErr.ExitCode())
		}
	}
}

func TestDelay(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	start := time.Now()
	cmd := exec.Command(bin, "-d", "200ms", "3", "echo", "hello")
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		t.Fatalf("repeat: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected at least 400ms (2 delays), got %v", elapsed)
	}
}

func TestUntilSuccess(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "--until-success", "true")
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		t.Fatalf("repeat: %v", err)
	}
}

func TestInvalidN(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "abc", "echo")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for invalid N")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
		}
	}
}

func TestMissingCommand(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "3")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestSymlink(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "1", "echo", "first")
	cmd.Dir = dir
	cmd.Run()

	time.Sleep(1 * time.Second)

	cmd = exec.Command(bin, "1", "echo", "second")
	cmd.Dir = dir
	cmd.Run()

	linkPath := filepath.Join(dir, ".repeat", "last")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target == "" {
		t.Error("symlink target is empty")
	}

	entries, _ := os.ReadDir(filepath.Join(dir, ".repeat"))
	sessionDirs := 0
	for _, e := range entries {
		if e.IsDir() {
			sessionDirs++
		}
	}
	if sessionDirs != 2 {
		t.Errorf("expected 2 session dirs, got %d", sessionDirs)
	}
}

func TestRunLastSymlink(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(bin, "3", "echo", "hello")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repeat: %v\n%s", err, out)
	}

	linkPath := filepath.Join(dir, ".repeat", "last", "last")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("run last symlink: %v", err)
	}
	if target != "run.3.log" {
		t.Errorf("expected run.3.log, got %s", target)
	}
}

func TestVersion(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repeat --version: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "repeat") {
		t.Errorf("expected version output to contain 'repeat', got %q", string(out))
	}
}

func TestUntilSuccessWithN(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--until-success", "3", "echo", "hello")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for --until-success with N")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
		}
	}
}

func countRunLogs(dir string) int {
	entries, err := os.ReadDir(filepath.Join(dir, ".repeat"))
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		files, _ := filepath.Glob(filepath.Join(dir, ".repeat", e.Name(), "run.*.log"))
		if len(files) > count {
			count = len(files)
		}
	}
	return count
}

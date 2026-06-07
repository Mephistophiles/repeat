package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mephistophiles/repeat/internal/runner"
)

func TestNewSession(t *testing.T) {
	dir := t.TempDir()

	s, err := NewSessionAt(dir)
	if err != nil {
		t.Fatalf("NewSessionAt: %v", err)
	}

	if _, err := os.Stat(s.Dir); os.IsNotExist(err) {
		t.Error("session dir not created")
	}

	linkPath := filepath.Join(s.BaseDir, "last")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != s.Dir {
		t.Errorf("symlink target mismatch: %s != %s", target, s.Dir)
	}
}

func TestCreateRunFile(t *testing.T) {
	dir := t.TempDir()

	s, err := NewSessionAt(dir)
	if err != nil {
		t.Fatalf("NewSessionAt: %v", err)
	}

	f, err := s.CreateRunFile(1)
	if err != nil {
		t.Fatalf("CreateRunFile: %v", err)
	}
	defer f.Close()

	expected := filepath.Join(s.Dir, "run.1.log")
	if f.Name() != expected {
		t.Errorf("expected %s, got %s", expected, f.Name())
	}
}

func TestWriteHeaderAndFooter(t *testing.T) {
	dir := t.TempDir()

	s, err := NewSessionAt(dir)
	if err != nil {
		t.Fatalf("NewSessionAt: %v", err)
	}

	f, err := s.CreateRunFile(3)
	if err != nil {
		t.Fatalf("CreateRunFile: %v", err)
	}
	defer f.Close()

	started := time.Now()
	WriteHeader(f, 3, 5, "echo hello", started)
	f.WriteString("hello\n")

	r := runner.Result{
		RunIndex:   3,
		ExitCode:   0,
		StartedAt:  started,
		FinishedAt: started.Add(10 * time.Millisecond),
		Duration:   10 * time.Millisecond,
	}
	WriteFooter(f, r)

	f.Seek(0, 0)
	data, _ := os.ReadFile(f.Name())
	content := string(data)

	if !strings.Contains(content, "# repeat — run 3 of 5") {
		t.Error("missing header: run info")
	}
	if !strings.Contains(content, "# command: echo hello") {
		t.Error("missing header: command")
	}
	if !strings.Contains(content, "---") {
		t.Error("missing separator")
	}
	if !strings.Contains(content, "hello") {
		t.Error("missing output")
	}
	if !strings.Contains(content, "# exit code: 0") {
		t.Error("missing footer: exit code")
	}
}

func TestSymlinkUpdated(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewSessionAt(dir)
	if err != nil {
		t.Fatalf("first NewSessionAt: %v", err)
	}

	time.Sleep(1 * time.Second)

	s2, err := NewSessionAt(dir)
	if err != nil {
		t.Fatalf("second NewSessionAt: %v", err)
	}

	linkPath := filepath.Join(s2.BaseDir, "last")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != s2.Dir {
		t.Errorf("symlink should point to s2 (%s), got %s (s1 was %s)", s2.Dir, target, s1.Dir)
	}
}

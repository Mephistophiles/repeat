package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Mephistophiles/repeat/internal/runner"
)

type Session struct {
	Dir     string
	BaseDir string
}

func NewSession() (*Session, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	return NewSessionAt(wd)
}

func NewSessionAt(workDir string) (*Session, error) {
	baseDir := filepath.Join(workDir, ".repeat")
	ts := time.Now().Format("2006-01-02T15-04-05")
	sessionDir := filepath.Join(baseDir, ts)

	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir session: %w", err)
	}

	linkPath := filepath.Join(baseDir, "last")
	_ = os.Remove(linkPath)
	if err := os.Symlink(sessionDir, linkPath); err != nil {
		return nil, fmt.Errorf("symlink last: %w", err)
	}

	return &Session{
		Dir:     sessionDir,
		BaseDir: baseDir,
	}, nil
}

func (s *Session) CreateRunFile(index int) (*os.File, error) {
	name := fmt.Sprintf("run.%d.log", index)
	path := filepath.Join(s.Dir, name)
	return os.Create(path)
}

func (s *Session) UpdateRunSymlink(index int) error {
	name := fmt.Sprintf("run.%d.log", index)
	linkPath := filepath.Join(s.Dir, "last")
	_ = os.Remove(linkPath)
	return os.Symlink(name, linkPath)
}

func WriteHeader(w *os.File, index, total int, command string, started time.Time) {
	if total > 0 {
		fmt.Fprintf(w, "# repeat — run %d of %d\n", index, total)
	} else {
		fmt.Fprintf(w, "# repeat — run %d\n", index)
	}
	fmt.Fprintf(w, "# command: %s\n", command)
	fmt.Fprintf(w, "# started:  %s\n", started.Format(time.RFC3339Nano))
	fmt.Fprintln(w, "---")
}

func WriteFooter(w *os.File, r runner.Result) {
	fmt.Fprintf(w, "\n---\n")
	fmt.Fprintf(w, "# finished: %s\n", r.FinishedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(w, "# duration: %s\n", r.Duration)
	if r.StdoutTruncated {
		fmt.Fprintf(w, "# stdout truncated at 64KB\n")
	}
	if r.StderrTruncated {
		fmt.Fprintf(w, "# stderr truncated at 64KB\n")
	}
	if r.TimedOut {
		fmt.Fprintf(w, "# exit code: - (timed out)\n")
	} else if r.Interrupted {
		fmt.Fprintf(w, "# exit code: - (interrupted)\n")
	} else {
		fmt.Fprintf(w, "# exit code: %d\n", r.ExitCode)
	}
}

package runner

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"time"
)

const maxBufSize = 64 * 1024

type Opts struct {
	Index    int
	Command  string
	Args     []string
	Timeout  time.Duration
	LogW     io.Writer
	LogWErr  io.Writer
	OnOutput func(string)
}

func Run(ctx context.Context, opts Opts) Result {
	fullCmd := opts.Command
	if len(opts.Args) > 0 {
		fullCmd += " " + strings.Join(opts.Args, " ")
	}

	r := Result{
		RunIndex:  opts.Index,
		Command:   fullCmd,
		StartedAt: time.Now(),
	}

	var runCtx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	} else {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, opts.Command, opts.Args...)

	var outBuf, errBuf bytes.Buffer

	writers := []io.Writer{&outBuf}
	if opts.LogW != nil {
		writers = append(writers, opts.LogW)
	}
	outW := io.MultiWriter(writers...)

	errWriters := []io.Writer{&errBuf}
	if opts.LogWErr != nil {
		errWriters = append(errWriters, opts.LogWErr)
	}
	errW := io.MultiWriter(errWriters...)

	var lwOut, lwErr *linedWriter
	if opts.OnOutput != nil {
		lwOut = &linedWriter{fn: opts.OnOutput}
		outW = io.MultiWriter(outW, lwOut)
		lwErr = &linedWriter{fn: opts.OnOutput}
		errW = io.MultiWriter(errW, lwErr)
	}

	cmd.Stdout = outW
	cmd.Stderr = errW

	err := cmd.Run()

	r.FinishedAt = time.Now()
	r.Duration = r.FinishedAt.Sub(r.StartedAt)
	r.DurationMs = r.Duration.Milliseconds()

	if lwOut != nil {
		lwOut.flush()
	}
	if lwErr != nil {
		lwErr.flush()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			r.ExitCode = exitErr.ExitCode()
		}
		select {
		case <-runCtx.Done():
			if runCtx.Err() == context.DeadlineExceeded {
				r.TimedOut = true
			} else {
				r.Interrupted = true
			}
		default:
		}
	}

	r.Stdout, r.StdoutTruncated = trim(outBuf.Bytes())
	r.Stderr, r.StderrTruncated = trim(errBuf.Bytes())

	return r
}

func trim(b []byte) (string, bool) {
	if len(b) > maxBufSize {
		return string(b[:maxBufSize]), true
	}
	return string(b), false
}

type linedWriter struct {
	fn  func(string)
	buf []byte
}

func (w *linedWriter) Write(p []byte) (int, error) {
	if w.fn == nil {
		return len(p), nil
	}
	for _, b := range p {
		if b == '\n' {
			w.fn(string(w.buf))
			w.buf = w.buf[:0]
		} else {
			w.buf = append(w.buf, b)
		}
	}
	return len(p), nil
}

func (w *linedWriter) flush() {
	if w.fn != nil && len(w.buf) > 0 {
		w.fn(string(w.buf))
		w.buf = w.buf[:0]
	}
}

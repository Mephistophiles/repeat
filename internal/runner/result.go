package runner

import "time"

type Result struct {
	RunIndex         int           `json:"run"`
	Command          string        `json:"command"`
	ExitCode         int           `json:"exit_code"`
	StartedAt        time.Time     `json:"started_at"`
	FinishedAt       time.Time     `json:"finished_at"`
	Duration         time.Duration `json:"-"`
	DurationMs       int64         `json:"duration_ms"`
	Stdout           string        `json:"stdout"`
	Stderr           string        `json:"stderr"`
	StdoutTruncated  bool          `json:"stdout_truncated"`
	StderrTruncated  bool          `json:"stderr_truncated"`
	Interrupted      bool          `json:"interrupted"`
	TimedOut         bool          `json:"timed_out"`
}

func (r *Result) OK() bool {
	return r.ExitCode == 0 && !r.Interrupted && !r.TimedOut
}

func (r *Result) ExitStatus() int {
	if r.TimedOut {
		return 124
	}
	if r.Interrupted {
		return 130
	}
	return r.ExitCode
}

func LastErrorExitCode(results []Result) int {
	for i := len(results) - 1; i >= 0; i-- {
		if !results[i].OK() {
			return results[i].ExitStatus()
		}
	}
	return 0
}

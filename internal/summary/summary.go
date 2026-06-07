package summary

import (
	"encoding/json"
	"io"

	"github.com/Mephistophiles/repeat/internal/runner"
)

type Output struct {
	Command         string          `json:"command"`
	Runs            []runner.Result `json:"runs"`
	Total           int             `json:"total"`
	Successes       int             `json:"successes"`
	Failures        int             `json:"failures"`
	TotalDurationMs int64           `json:"total_duration_ms"`
}

func Build(command string, results []runner.Result) Output {
	var successes, failures int
	var totalMs int64
	for _, r := range results {
		totalMs += r.DurationMs
		if r.OK() {
			successes++
		} else {
			failures++
		}
	}
	return Output{
		Command:         command,
		Runs:            results,
		Total:           len(results),
		Successes:       successes,
		Failures:        failures,
		TotalDurationMs: totalMs,
	}
}

func Write(w io.Writer, o Output) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}

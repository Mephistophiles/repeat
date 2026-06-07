package tui

import (
	"testing"
	"time"

	"github.com/Mephistophiles/repeat/internal/runner"
)

func TestProgressFraction(t *testing.T) {
	m := Model{Total: 10}
	if m.ProgressFraction() != 0 {
		t.Errorf("expected 0, got %f", m.ProgressFraction())
	}

	m.Results = append(m.Results, runner.Result{})
	if m.ProgressFraction() != 0.1 {
		t.Errorf("expected 0.1, got %f", m.ProgressFraction())
	}

	m.Results = append(m.Results, runner.Result{}, runner.Result{}, runner.Result{})
	if m.ProgressFraction() != 0.4 {
		t.Errorf("expected 0.4, got %f", m.ProgressFraction())
	}
}

func TestProgressFractionZero(t *testing.T) {
	m := Model{Total: 0}
	if m.ProgressFraction() != 0 {
		t.Errorf("expected 0 for zero total, got %f", m.ProgressFraction())
	}
}

func TestETA(t *testing.T) {
	m := Model{
		Total: 5,
		Results: []runner.Result{
			{Duration: 100 * time.Millisecond},
			{Duration: 200 * time.Millisecond},
		},
		Delay: 50 * time.Millisecond,
	}

	eta := m.ETA()
	expectedAvg := 150 * time.Millisecond
	remaining := 3
	expectedETA := expectedAvg*time.Duration(remaining) + 50*time.Millisecond*time.Duration(remaining)

	if eta != expectedETA {
		t.Errorf("expected ETA %v, got %v", expectedETA, eta)
	}
}

func TestETANoResults(t *testing.T) {
	m := Model{Total: 5}
	if m.ETA() != 0 {
		t.Errorf("expected ETA 0 with no results, got %v", m.ETA())
	}
}

func TestAvgRun(t *testing.T) {
	m := Model{
		Results: []runner.Result{
			{Duration: 100 * time.Millisecond},
			{Duration: 200 * time.Millisecond},
			{Duration: 300 * time.Millisecond},
		},
	}

	avg := m.avgRun()
	if avg != 200*time.Millisecond {
		t.Errorf("expected 200ms avg, got %v", avg)
	}
}

func TestAvgRunEmpty(t *testing.T) {
	m := Model{}
	if m.avgRun() != 0 {
		t.Errorf("expected 0 avg for no results, got %v", m.avgRun())
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{100 * time.Millisecond, "100ms"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1m 30s"},
		{2 * time.Hour, "2h 0m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

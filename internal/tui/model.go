package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/Mephistophiles/repeat/internal/runner"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeUntilSuccess
)

type Model struct {
	Mode    Mode
	Total   int
	Command string

	Results    []runner.Result
	CurrentRun int
	Running    bool
	Finished   bool
	StartTime  time.Time

	Progress    progress.Model
	Spinner     spinner.Model
	Verbose     bool
	Delay       time.Duration
	StopOnError bool

	OutputLines []string

	MsgChan chan Msg
	Done    chan struct{}

	Width  int
	Height int
}

type Config struct {
	Mode        Mode
	Total       int
	Command     string
	Verbose     bool
	Delay       time.Duration
	StopOnError bool
}

type RunnerFunc func(msgChan chan<- Msg, done <-chan struct{})

type Msg interface{}

type RunStartedMsg struct{ Index int }
type RunOutputMsg struct{ Index int; Line string }
type RunCompletedMsg struct{ Result runner.Result }
type InterruptedMsg struct{}

func NewModel(cfg Config) Model {
	p := progress.New(progress.WithDefaultGradient())
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		Mode:        cfg.Mode,
		Total:       cfg.Total,
		Command:     cfg.Command,
		Progress:    p,
		Spinner:     s,
		Verbose:     cfg.Verbose,
		Delay:       cfg.Delay,
		StopOnError: cfg.StopOnError,
		MsgChan:     make(chan Msg, 64),
		Done:        make(chan struct{}),
	}
}

func (m Model) ETA() time.Duration {
	completed := len(m.Results)
	if completed == 0 {
		return 0
	}
	var totalDur time.Duration
	for _, r := range m.Results {
		totalDur += r.Duration
	}
	avg := totalDur / time.Duration(completed)
	remaining := m.Total - completed
	if remaining < 0 {
		remaining = 0
	}
	return avg*time.Duration(remaining) + m.Delay*time.Duration(remaining)
}

func (m Model) ProgressFraction() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(len(m.Results)) / float64(m.Total)
}

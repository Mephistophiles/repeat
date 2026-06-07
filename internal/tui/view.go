package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/Mephistophiles/repeat/internal/runner"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	etaStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	runStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	sepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m Model) View() string {
	if m.Mode == ModeUntilSuccess {
		return m.viewUntilSuccess()
	}
	return m.viewNormal()
}

func (m Model) viewNormal() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("repeat: %s", m.Command)))
	b.WriteString("\n\n")

	frac := m.ProgressFraction()
	b.WriteString(m.Progress.ViewAs(frac))
	b.WriteString(fmt.Sprintf("  %d/%d  %d%%\n", len(m.Results), m.Total, int(frac*100)))
	b.WriteString("\n")

	if m.Finished {
		b.WriteString(etaStyle.Render("Done."))
	} else {
		eta := m.ETA()
		b.WriteString(etaStyle.Render(fmt.Sprintf("ETA: ~%s (avg %s/run)", formatDuration(eta), formatDuration(m.avgRun()))))
	}
	b.WriteString("\n\n")

	start := len(m.Results) - 3
	if start < 0 {
		start = 0
	}
	for _, r := range m.Results[start:] {
		b.WriteString(formatResult(r))
		b.WriteString("\n")
	}

	if m.Running {
		b.WriteString(runStyle.Render(fmt.Sprintf("> Run %d: running...", m.CurrentRun)))
		b.WriteString("\n")
	}

	if m.Verbose && len(m.OutputLines) > 0 {
		b.WriteString("\n")
		w := m.Width
		if w == 0 {
			w = 80
		}
		b.WriteString(sepStyle.Render(strings.Repeat("─", w)))
		b.WriteString("\n")

		availableLines := m.Height - strings.Count(b.String(), "\n") - 3
		if availableLines < 3 {
			availableLines = 3
		}

		start := len(m.OutputLines) - availableLines
		if start < 0 {
			start = 0
		}
		for _, line := range m.OutputLines[start:] {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) viewUntilSuccess() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("repeat: %s", m.Command)))
	b.WriteString("\n\n")

	b.WriteString(m.Spinner.View())
	b.WriteString(fmt.Sprintf(" Attempt %d\n", len(m.Results)+1))
	b.WriteString("\n")

	elapsed := time.Duration(0)
	for _, r := range m.Results {
		elapsed += r.Duration
	}
	b.WriteString(etaStyle.Render(fmt.Sprintf("Elapsed: %s", formatDuration(elapsed))))
	b.WriteString("\n\n")

	if len(m.Results) > 0 {
		last := m.Results[len(m.Results)-1]
		b.WriteString(formatResult(last))
	}

	return b.String()
}

func formatResult(r runner.Result) string {
	status := "ok"
	style := runStyle
	if r.TimedOut {
		status = "timed out"
		style = errStyle
	} else if r.Interrupted {
		status = "interrupted"
		style = errStyle
	} else if r.ExitCode != 0 {
		status = fmt.Sprintf("exit %d", r.ExitCode)
		style = errStyle
	}
	return style.Render(fmt.Sprintf("> Run %d: %s (%s)", r.RunIndex, status, formatDuration(r.Duration)))
}

func (m Model) avgRun() time.Duration {
	if len(m.Results) == 0 {
		return 0
	}
	var total time.Duration
	for _, r := range m.Results {
		total += r.Duration
	}
	return total / time.Duration(len(m.Results))
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := d / time.Minute
		s := (d % time.Minute).Seconds()
		return fmt.Sprintf("%dm %.0fs", m, s)
	}
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	return fmt.Sprintf("%dh %dm", h, m)
}

package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.listen(),
		m.Spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			close(m.Done)
			return m, tea.Quit
		}

	case RunStartedMsg:
		m.CurrentRun = msg.Index
		m.Running = true
		return m, nil

	case RunOutputMsg:
		if m.Verbose {
			m.OutputLines = append(m.OutputLines, msg.Line)
		}
		return m, nil

	case RunCompletedMsg:
		res := msg.Result
		m.Results = append(m.Results, res)
		m.Running = false

		if m.StopOnError && !res.OK() {
			m.Finished = true
			return m, tea.Quit
		}

		if m.Mode == ModeUntilSuccess && res.OK() {
			m.Finished = true
			return m, tea.Quit
		}

		if m.Mode == ModeNormal && m.Total > 0 && len(m.Results) >= m.Total {
			m.Finished = true
			return m, tea.Quit
		}

		return m, m.listen()

	case InterruptedMsg:
		m.Finished = true
		return m, tea.Quit

	case tickMsg:
		if !m.Finished {
			return m, m.listenTick()
		}

	default:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) listen() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-m.MsgChan:
			if !ok {
				return InterruptedMsg{}
			}
			return msg
		case <-m.Done:
			return InterruptedMsg{}
		}
	}
}

func (m Model) listenTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		select {
		case msg, ok := <-m.MsgChan:
			if !ok {
				return InterruptedMsg{}
			}
			return msg
		case <-m.Done:
			return InterruptedMsg{}
		case <-time.After(100 * time.Millisecond):
			return tickMsg(t)
		}
	})
}

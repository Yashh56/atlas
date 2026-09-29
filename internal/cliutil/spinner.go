package cliutil

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerModel struct {
	spinner   spinner.Model
	message   string
	startTime time.Time
	done      bool
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case spinnerUpdateMsg:
		m.message = string(msg)
		return m, nil
	case spinnerStopMsg:
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	elapsed := time.Since(m.startTime).Seconds()
	timeStr := StyleMuted.Render(fmt.Sprintf("(%.1fs)", elapsed))
	return fmt.Sprintf("%s %s %s", m.spinner.View(), StyleBody.Render(m.message), timeStr)
}

type spinnerUpdateMsg string
type spinnerStopMsg struct{}

// Spinner represents a CLI loader built on Bubble Tea.
type Spinner struct {
	prog *tea.Program
}

// StartSpinner starts a loader with a message.
func StartSpinner(message string) *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StylePrimary

	m := spinnerModel{
		spinner:   s,
		message:   message,
		startTime: time.Now(),
	}

	p := tea.NewProgram(m)
	go p.Run()

	return &Spinner{prog: p}
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.prog.Send(spinnerStopMsg{})
	// Wait a tiny bit for UI to clear
	time.Sleep(50 * time.Millisecond) 
}

// UpdateMessage changes the spinner message.
func (s *Spinner) UpdateMessage(newMessage string) {
	s.prog.Send(spinnerUpdateMsg(newMessage))
}

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type wizardStep int

const (
	stepLoading wizardStep = iota
	stepModelSelect
	stepModelInput
	stepActionSelect
	stepProviderSelect
	stepVercelConfirm
	stepVercelInput
	stepDone
)

type wizardModel struct {
	step wizardStep
	err  error

	store *credentials.Store

	selectedModel    string
	selectedAction   orchestrator.Action
	selectedProvider string

	width  int
	height int

	spinner   spinner.Model
	list      list.Model
	textInput textinput.Model
}

type initCompleteMsg struct{}
type advanceStepMsg struct{}

func RunWizard(modelFlag string, actionFlag orchestrator.Action, providerFlag string) (string, orchestrator.Action, string, error) {
	store, _ := credentials.Open()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := wizardModel{
		step:             stepLoading,
		store:            store,
		spinner:          s,
		selectedModel:    modelFlag,
		selectedAction:   actionFlag,
		selectedProvider: providerFlag,
	}

	m.textInput = textinput.New()
	m.textInput.EchoMode = textinput.EchoPassword
	m.textInput.Focus()

	p := tea.NewProgram(&m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", "", "", err
	}

	res := finalModel.(*wizardModel)
	if res.err != nil {
		return "", "", "", res.err
	}
	if res.step != stepDone {
		return "", "", "", fmt.Errorf("wizard aborted")
	}

	return res.selectedModel, res.selectedAction, res.selectedProvider, nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// -- Bubble Tea Interface Methods --

func (m *wizardModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		func() tea.Msg {
			importTime := 600 * 1000 * 1000 // 600ms
			for i := 0; i < importTime; i++ {
			}
			return initCompleteMsg{}
		},
	)
}

func (m *wizardModel) nextStep() tea.Cmd {
	if m.selectedModel == "" {
		m.step = stepModelSelect
		m.list = createModelList(m.store)
		if m.width > 0 && m.height > 0 {
			m.list.SetSize(m.width, m.height)
		}
		return nil
	}
	if m.selectedAction == "" {
		m.step = stepActionSelect
		m.list = createActionList()
		if m.width > 0 && m.height > 0 {
			m.list.SetSize(m.width, m.height)
		}
		return nil
	}
	if actionNeedsProvider(m.selectedAction) && m.selectedProvider == "" {
		m.step = stepProviderSelect
		m.list = createProviderList(m.store)
		if m.width > 0 && m.height > 0 {
			m.list.SetSize(m.width, m.height)
		}
		return nil
	}
	m.step = stepDone
	return tea.Quit
}

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width = msg.Width - h
		m.height = msg.Height - v
		if m.step == stepModelSelect || m.step == stepActionSelect || m.step == stepProviderSelect {
			m.list.SetSize(m.width, m.height)
		}
	case initCompleteMsg:
		if m.step == stepLoading {
			return m, m.nextStep()
		}
	case advanceStepMsg:
		return m, m.nextStep()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.step {
	case stepModelSelect:
		return m.updateModelSelect(msg)
	case stepModelInput:
		return m.updateModelInput(msg)
	case stepActionSelect:
		return m.updateActionSelect(msg)
	case stepProviderSelect:
		return m.updateProviderSelect(msg)
	case stepVercelConfirm:
		return m.updateVercelConfirm(msg)
	case stepVercelInput:
		return m.updateVercelInput(msg)
	}
	return m, nil
}

func (m *wizardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.step {
	case stepLoading:
		return fmt.Sprintf("\n\n   %s Initializing Atlas interactive wizard...\n\n", m.spinner.View())
	case stepModelSelect:
		return docStyle.Render(m.list.View())
	case stepModelInput:
		return fmt.Sprintf(
			"Enter API key for %s:\n\n%s\n\n(esc to abort)",
			m.selectedModel,
			m.textInput.View(),
		)
	case stepActionSelect:
		return docStyle.Render(m.list.View())
	case stepProviderSelect:
		return docStyle.Render(m.list.View())
	case stepVercelConfirm:
		return "No Vercel token found. Create one at https://vercel.com/account/tokens\nOpen in browser now? [y/N]: "
	case stepVercelInput:
		return fmt.Sprintf(
			"Enter Vercel token:\n\n%s\n\n(esc to abort)",
			m.textInput.View(),
		)
	case stepDone:
		return "Wizard complete. Launching...\n"
	}
	return ""
}

// -- Step Handlers --

func (m *wizardModel) updateModelSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selectedModel = i.title
				if strings.Contains(i.desc, "✓") || m.selectedModel == "local" {
					return m, func() tea.Msg { return advanceStepMsg{} }
				}
				m.step = stepModelInput
				m.textInput.Reset()
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *wizardModel) updateModelInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			key := strings.TrimSpace(m.textInput.Value())
			if key != "" && m.store != nil {
				m.store.SetSecret("llm:"+m.selectedModel, key)
				m.store.SetMeta(credentials.ProviderCredential{
					Provider:   m.selectedModel,
					Method:     credentials.MethodStoredToken,
				})
			}
			return m, func() tea.Msg { return advanceStepMsg{} }
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *wizardModel) updateActionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				switch i.title {
				case "Just build":
					m.selectedAction = orchestrator.ActionBuild
				case "Build + test":
					m.selectedAction = orchestrator.ActionTest
				case "Deploy":
					m.selectedAction = orchestrator.ActionDeploy
				case "Test + deploy":
					m.selectedAction = orchestrator.ActionTestAndDeploy
				}
				return m, func() tea.Msg { return advanceStepMsg{} }
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *wizardModel) updateProviderSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				if i.title != "vercel" {
					return m, nil
				}
				m.selectedProvider = i.title
				if strings.Contains(i.desc, "✓") {
					return m, func() tea.Msg { return advanceStepMsg{} }
				}
				m.step = stepVercelConfirm
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *wizardModel) updateVercelConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		if s == "y" || s == "Y" {
			_ = openBrowser("https://vercel.com/account/tokens")
			m.step = stepVercelInput
			m.textInput.Reset()
			return m, nil
		} else if s == "n" || s == "N" || s == "enter" {
			m.step = stepVercelInput
			m.textInput.Reset()
			return m, nil
		}
	}
	return m, nil
}

func (m *wizardModel) updateVercelInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			key := strings.TrimSpace(m.textInput.Value())
			if key != "" && m.store != nil {
				m.store.SetSecret("vercel", key)
				m.store.SetMeta(credentials.ProviderCredential{
					Provider: "vercel",
					Method:   credentials.MethodStoredToken,
				})
			}
			return m, func() tea.Msg { return advanceStepMsg{} }
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// -- UI Helpers --

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func createModelList(store *credentials.Store) list.Model {
	providers := []string{"anthropic", "openai", "gemini", "mistral", "groq", "grok", "local"}
	items := make([]list.Item, 0, len(providers))

	for _, p := range providers {
		status := "✗ not configured"
		if p == "local" {
			status = "no key needed"
		} else {
			if os.Getenv(strings.ToUpper(p)+"_API_KEY") != "" {
				status = "✓ ENV var detected"
			} else if store != nil {
				if s, _ := store.GetSecret("llm:" + p); s != "" {
					status = "✓ stored"
				}
			}
		}
		items = append(items, item{title: p, desc: status})
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 15)
	l.Title = "Select LLM Provider"
	l.SetShowStatusBar(false)
	return l
}

func createProviderList(store *credentials.Store) list.Model {
	providers := []string{"vercel", "render", "netlify", "fly", "railway"}
	items := make([]list.Item, 0, len(providers))

	for _, p := range providers {
		status := "✗ not configured"
		if p == "vercel" {
			if os.Getenv("VERCEL_TOKEN") != "" {
				status = "✓ ENV var detected"
			} else if store != nil {
				if s, _ := store.GetSecret("vercel"); s != "" {
					status = "✓ stored"
				}
			}
		} else {
			status = "not implemented yet"
		}
		items = append(items, item{title: p, desc: status})
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 15)
	l.Title = "Select Deploy Provider"
	l.SetShowStatusBar(false)
	return l
}

func createActionList() list.Model {
	items := []list.Item{
		item{title: "Just build", desc: "analyze, validate, build"},
		item{title: "Build + test", desc: "build, then run tests"},
		item{title: "Deploy", desc: "build, fix if needed, and deploy (default)"},
		item{title: "Test + deploy", desc: "build, fix, run tests, and deploy"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 15)
	l.Title = "Select Action Mode"
	l.SetShowStatusBar(false)
	return l
}

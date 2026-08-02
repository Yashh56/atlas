package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"context"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/llm"
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
	stepLocalModelLoading
	stepLocalModelSelect
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

type localModelsFetchedMsg struct {
	models []string
	err    error
}

func fetchLocalModelsCmd() tea.Cmd {
	return func() tea.Msg {
		configPath := filepath.Join(".atlas", "config.json")
		cfg, err := config.Load(configPath)
		baseURL := "http://localhost:11434/v1"
		if err == nil && cfg.LocalLLMBaseURL != "" {
			baseURL = cfg.LocalLLMBaseURL
		}
		lister, ok := llm.NewModelLister("local", "", baseURL)
		if !ok {
			return localModelsFetchedMsg{err: fmt.Errorf("local model lister not available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := lister.ListModels(ctx)
		return localModelsFetchedMsg{models: models, err: err}
	}
}

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
		if m.step == stepModelSelect || m.step == stepActionSelect || m.step == stepProviderSelect || m.step == stepLocalModelSelect {
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
	case localModelsFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		if len(msg.models) == 0 {
			m.err = fmt.Errorf("No models found — pull one first, e.g. 'ollama pull qwen2.5-coder:7b'")
			return m, tea.Quit
		}
		var items []list.Item
		for _, model := range msg.models {
			items = append(items, item{title: model, desc: "local model"})
		}
		m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height)
		m.list.Title = "Select Local Model"
		m.list.SetShowStatusBar(false)
		m.step = stepLocalModelSelect
		return m, nil
	}

	switch m.step {
	case stepModelSelect:
		return m.updateModelSelect(msg)
	case stepLocalModelSelect:
		return m.updateLocalModelSelect(msg)
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
	case stepLocalModelLoading:
		return fmt.Sprintf("\n\n   %s Discovering local models...\n\n", m.spinner.View())
	case stepModelSelect, stepLocalModelSelect:
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
				if m.selectedModel == "local" {
					m.step = stepLocalModelLoading
					return m, fetchLocalModelsCmd()
				}
				if strings.Contains(i.desc, "✓") {
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

func (m *wizardModel) updateLocalModelSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.selectedModel = i.title
				configPath := filepath.Join(".atlas", "config.json")
				if cfg, err := config.Load(configPath); err == nil {
					cfg.LLMProvider = "local"
					cfg.DefaultModel = m.selectedModel
					_ = cfg.Save(configPath)
				}
				return m, func() tea.Msg { return advanceStepMsg{} }
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
				provider := getProviderForModel(m.selectedModel)
				m.store.SetSecret("llm:"+provider, key)
				m.store.SetMeta(credentials.ProviderCredential{
					Provider:   provider,
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
				if i.title != "vercel" && i.title != "render" && i.title != "netlify" {
					return m, nil
				}
				m.selectedProvider = i.title
				if strings.Contains(i.desc, "✓") {
					return m, func() tea.Msg { return advanceStepMsg{} }
				}
				if i.title == "vercel" {
					m.step = stepVercelConfirm
					return m, nil
				}
				// For render and netlify, if not configured, we just advance and let the CLI prompt.
				return m, func() tea.Msg { return advanceStepMsg{} }
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
	var items []list.Item

	for _, p := range llmProviderEnvVars {
		status := "✗ not configured"
		if p.Name == "local" {
			status = "no key needed"
		} else {
			if os.Getenv(p.EnvVar) != "" {
				status = "✓ ENV var detected"
			} else if store != nil {
				if s, _ := store.GetSecret("llm:" + p.Name); s != "" {
					status = "✓ stored"
				}
			}
		}

		if p.Name == "local" {
			items = append(items, item{title: "local", desc: "local (discover models dynamically)"})
		} else {
			for _, m := range p.Models {
				items = append(items, item{title: m, desc: fmt.Sprintf("%s (%s)", p.Name, status)})
			}
		}
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
		var envVar string
		if p == "vercel" {
			envVar = "VERCEL_TOKEN"
		} else if p == "render" {
			envVar = "RENDER_TOKEN"
		} else if p == "netlify" {
			envVar = "NETLIFY_TOKEN"
		}

		if envVar != "" && os.Getenv(envVar) != "" {
			status = "✓ ENV var detected"
		} else if envVar != "" && store != nil {
			if meta, ok, _ := store.GetMeta(p); ok {
				if meta.Method == credentials.MethodStoredToken {
					status = "✓ stored"
				} else if meta.Method == credentials.MethodCLISession {
					status = "✓ CLI logged in"
				}
			}
		} else if envVar == "" {
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

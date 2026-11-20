package cmd

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWizard_RailwayInProviderList(t *testing.T) {
	// Without env var set
	_ = os.Unsetenv("RAILWAY_API_TOKEN")
	_ = os.Unsetenv("RAILWAY_TOKEN")

	l := createProviderList(nil)
	items := l.Items()

	found := false
	for _, it := range items {
		if i, ok := it.(item); ok && i.title == "railway" {
			found = true
			if i.desc != "✗ not configured" {
				t.Errorf("expected '✗ not configured', got %q", i.desc)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected railway to be in provider list, but not found")
	}

	// With RAILWAY_API_TOKEN set
	t.Setenv("RAILWAY_API_TOKEN", "test-token")
	l = createProviderList(nil)
	found = false
	for _, it := range l.Items() {
		if i, ok := it.(item); ok && i.title == "railway" {
			found = true
			if i.desc != "✓ ENV var detected" {
				t.Errorf("expected '✓ ENV var detected', got %q", i.desc)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected railway in provider list")
	}

	// With RAILWAY_TOKEN set
	_ = os.Unsetenv("RAILWAY_API_TOKEN")
	t.Setenv("RAILWAY_TOKEN", "test-token-alt")
	l = createProviderList(nil)
	found = false
	for _, it := range l.Items() {
		if i, ok := it.(item); ok && i.title == "railway" {
			found = true
			if i.desc != "✓ ENV var detected" {
				t.Errorf("expected '✓ ENV var detected' with RAILWAY_TOKEN, got %q", i.desc)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected railway in provider list")
	}
}

func TestWizard_UpdateProviderSelect_Railway(t *testing.T) {
	m := &wizardModel{
		step: stepProviderSelect,
		list: createProviderList(nil),
	}
	for idx, it := range m.list.Items() {
		if i, ok := it.(item); ok && i.title == "railway" {
			m.list.Select(idx)
			break
		}
	}

	newModel, cmd := m.updateProviderSelect(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Errorf("expected non-nil cmd advancing step")
	}
	wm, ok := newModel.(*wizardModel)
	if !ok {
		t.Fatalf("expected *wizardModel")
	}
	if wm.selectedProvider != "railway" {
		t.Errorf("expected selectedProvider == 'railway', got %q", wm.selectedProvider)
	}
}

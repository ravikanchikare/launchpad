package picker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectorArrowFilterAndSelect(t *testing.T) {
	model := newSelectorModel("Select", []Item{
		{Name: "claude-opus-5"},
		{Name: "claude-sonnet-5"},
		{Name: "gpt-5.6-sol"},
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(selectorModel)
	if model.cursor != 1 {
		t.Fatalf("cursor=%d", model.cursor)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("gpt")})
	model = updated.(selectorModel)
	if got := model.filteredItems(); len(got) != 1 || got[0].Name != "gpt-5.6-sol" {
		t.Fatalf("filtered=%#v", got)
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(selectorModel)
	if command == nil || model.selected != "gpt-5.6-sol" {
		t.Fatalf("selected=%q command=%v", model.selected, command)
	}
}

func TestSelectorCancel(t *testing.T) {
	model := newSelectorModel("Select", []Item{{Name: "model"}})
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(selectorModel)
	if command == nil || !model.cancelled {
		t.Fatalf("cancelled=%v command=%v", model.cancelled, command)
	}
}

func TestConfirmationNavigationAndLabels(t *testing.T) {
	model := newConfirmModel("Restart?", ConfirmOptions{
		YesLabel: "Restart now",
		NoLabel:  "Later",
		Default:  ConfirmDefaultYes,
	})
	if view := model.View(); !strings.Contains(view, "Restart now") || !strings.Contains(view, "Later") {
		t.Fatalf("view=%q", view)
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(confirmModel)
	if model.yes {
		t.Fatal("right arrow did not select No")
	}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(confirmModel)
	if command == nil || !model.confirmed || model.yes {
		t.Fatalf("confirmed=%v yes=%v command=%v", model.confirmed, model.yes, command)
	}
}

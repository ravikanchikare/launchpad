package picker

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"harnezpad/internal/gateway"
)

func TestModelPickerFiltersNavigatesAndSelects(t *testing.T) {
	picker := newModelPicker("Select model:", []gateway.Model{{ID: "kimi-k3"}, {ID: "claude-sonnet-5"}, {ID: "gpt-5.6-sol"}})
	updated, _ := picker.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("gpt")})
	picker = updated.(modelPicker)
	if len(picker.filtered) != 1 || picker.models[picker.filtered[0]].ID != "gpt-5.6-sol" {
		t.Fatalf("filtered models = %#v", picker.filtered)
	}
	updated, _ = picker.Update(tea.KeyMsg{Type: tea.KeyEnter})
	picker = updated.(modelPicker)
	if picker.selected != "gpt-5.6-sol" {
		t.Fatalf("selected = %q", picker.selected)
	}
}

func TestModelPickerBackspaceAndCancel(t *testing.T) {
	picker := newModelPicker("Select model:", []gateway.Model{{ID: "kimi-k3"}, {ID: "gpt-5.6-sol"}})
	updated, _ := picker.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	picker = updated.(modelPicker)
	if len(picker.filtered) != 0 || !strings.Contains(picker.View(), "no matches") {
		t.Fatalf("expected no matches: %s", picker.View())
	}
	updated, _ = picker.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	picker = updated.(modelPicker)
	if len(picker.filtered) != 2 {
		t.Fatalf("backspace did not restore models: %#v", picker.filtered)
	}
	updated, _ = picker.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !updated.(modelPicker).cancelled {
		t.Fatal("escape should cancel picker")
	}
}

func TestRestartConfirmationSelectsNo(t *testing.T) {
	confirmation := restartConfirmation{}
	updated, _ := confirmation.Update(tea.KeyMsg{Type: tea.KeyRight})
	confirmation = updated.(restartConfirmation)
	updated, _ = confirmation.Update(tea.KeyMsg{Type: tea.KeyEnter})
	confirmation = updated.(restartConfirmation)
	if !confirmation.selected || confirmation.choice != 1 {
		t.Fatalf("confirmation = %#v", confirmation)
	}
}

func TestRestartConfirmationEscapeCancels(t *testing.T) {
	updated, _ := (restartConfirmation{}).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !updated.(restartConfirmation).cancelled {
		t.Fatal("escape should cancel restart confirmation")
	}
}

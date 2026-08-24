package picker

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"harnezpad/internal/gateway"
)

const modelPickerPageSize = 12

type modelPicker struct {
	title     string
	models    []gateway.Model
	filtered  []int
	filter    string
	cursor    int
	selected  string
	cancelled bool
}

func newModelPicker(title string, models []gateway.Model) modelPicker {
	m := modelPicker{title: title, models: models}
	m.applyFilter()
	return m
}

func (m modelPicker) Init() tea.Cmd { return nil }

func (m modelPicker) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc, tea.KeyLeft:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyUp:
		if len(m.filtered) > 0 {
			m.cursor = (m.cursor - 1 + len(m.filtered)) % len(m.filtered)
		}
	case tea.KeyDown:
		if len(m.filtered) > 0 {
			m.cursor = (m.cursor + 1) % len(m.filtered)
		}
	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			m.selected = m.models[m.filtered[m.cursor]].ID
			return m, tea.Quit
		}
	case tea.KeyBackspace, tea.KeyDelete:
		if m.filter != "" {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.applyFilter()
		}
	case tea.KeyRunes:
		for _, r := range key.Runes {
			if unicode.IsPrint(r) {
				m.filter += string(r)
			}
		}
		m.applyFilter()
	case tea.KeySpace:
		m.filter += " "
		m.applyFilter()
	}
	return m, nil
}

func (m *modelPicker) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	m.filtered = m.filtered[:0]
	for i, model := range m.models {
		if query == "" || strings.Contains(strings.ToLower(model.ID), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
}

func (m modelPicker) View() string {
	if m.selected != "" || m.cancelled {
		return ""
	}
	var b strings.Builder
	b.WriteString("\033[1m" + m.title + "\033[0m ")
	if m.filter == "" {
		b.WriteString("\033[2mType to filter...\033[0m")
	} else {
		b.WriteString(m.filter)
	}
	b.WriteString("\n\n")
	if len(m.filtered) == 0 {
		b.WriteString("  \033[2m(no matches)\033[0m\n")
	} else {
		start := m.cursor - modelPickerPageSize/2
		if start < 0 {
			start = 0
		}
		if maxStart := len(m.filtered) - modelPickerPageSize; start > maxStart && maxStart > 0 {
			start = maxStart
		}
		end := start + modelPickerPageSize
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		if start > 0 {
			fmt.Fprintf(&b, "  \033[2m... %d more above\033[0m\n", start)
		}
		for position := start; position < end; position++ {
			name := m.models[m.filtered[position]].ID
			if position == m.cursor {
				b.WriteString("\033[7m  ▸ " + name + "  \033[0m\n")
			} else {
				b.WriteString("    " + name + "\n")
			}
		}
		if remaining := len(m.filtered) - end; remaining > 0 {
			fmt.Fprintf(&b, "  \033[2m... and %d more\033[0m\n", remaining)
		}
	}
	b.WriteString("\n\033[2m↑/↓ navigate • enter select • type to filter • esc cancel\033[0m")
	return b.String()
}

func Run(title string, models []gateway.Model) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return "", errors.New("model selection requires an interactive terminal; use --model <model>")
	}
	result, err := tea.NewProgram(newModelPicker(title, models), tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr), tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	picker, ok := result.(modelPicker)
	if !ok || picker.cancelled || picker.selected == "" {
		return "", errors.New("model selection cancelled")
	}
	return picker.selected, nil
}

type restartConfirmation struct {
	choice    int
	selected  bool
	cancelled bool
}

func (m restartConfirmation) Init() tea.Cmd { return nil }

func (m restartConfirmation) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyLeft, tea.KeyUp:
		m.choice = 0
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		m.choice = 1
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch unicode.ToLower(key.Runes[0]) {
			case 'y':
				m.choice, m.selected = 0, true
				return m, tea.Quit
			case 'n':
				m.choice, m.selected = 1, true
				return m, tea.Quit
			}
		}
	case tea.KeyEnter:
		m.selected = true
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

func (m restartConfirmation) View() string {
	if m.selected || m.cancelled {
		return ""
	}
	yes, no := " Yes ", " No "
	if m.choice == 0 {
		yes = "\033[7m" + yes + "\033[0m"
	} else {
		no = "\033[7m" + no + "\033[0m"
	}
	return "\033[1mRestart ChatGPT to use HarnezPad?\033[0m\n\n  " + yes + "    " + no + "\n\n\033[2m←/→ navigate • enter confirm • esc cancel\033[0m"
}

func ConfirmChatGPTRestart() (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return false, nil
	}
	result, err := tea.NewProgram(restartConfirmation{}, tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return false, err
	}
	confirmation, ok := result.(restartConfirmation)
	if !ok || confirmation.cancelled || !confirmation.selected {
		return false, nil
	}
	return confirmation.choice == 0, nil
}

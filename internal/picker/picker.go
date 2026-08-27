package picker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const pageSize = 10

var ErrCancelled = errors.New("selection cancelled")

type Item struct {
	Name        string
	Description string
}

type selectorModel struct {
	title     string
	items     []Item
	filter    string
	cursor    int
	offset    int
	selected  string
	cancelled bool
}

func newSelectorModel(title string, items []Item) selectorModel {
	return selectorModel{title: title, items: items}
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) filteredItems() []Item {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	if query == "" {
		return m.items
	}
	filtered := make([]Item, 0, len(m.items))
	for _, item := range m.items {
		if strings.Contains(strings.ToLower(item.Name+" "+item.Description), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m selectorModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	filtered := m.filteredItems()
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		if len(filtered) > 0 && m.cursor >= 0 && m.cursor < len(filtered) {
			m.selected = filtered[m.cursor].Name
			return m, tea.Quit
		}
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	case tea.KeyPgUp:
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
	case tea.KeyPgDown:
		m.cursor += pageSize
		if m.cursor >= len(filtered) {
			m.cursor = len(filtered) - 1
		}
	case tea.KeyBackspace, tea.KeyDelete:
		runes := []rune(m.filter)
		if len(runes) > 0 {
			m.filter = string(runes[:len(runes)-1])
			m.cursor, m.offset = 0, 0
		}
	case tea.KeyRunes:
		for _, r := range key.Runes {
			if unicode.IsPrint(r) {
				m.filter += string(r)
			}
		}
		m.cursor, m.offset = 0, 0
	case tea.KeySpace:
		m.filter += " "
		m.cursor, m.offset = 0, 0
	}
	filtered = m.filteredItems()
	if len(filtered) == 0 {
		m.cursor, m.offset = 0, 0
	} else {
		if m.cursor >= len(filtered) {
			m.cursor = len(filtered) - 1
		}
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
		if m.cursor >= m.offset+pageSize {
			m.offset = m.cursor - pageSize + 1
		}
	}
	return m, nil
}

func (m selectorModel) View() string {
	if m.selected != "" || m.cancelled {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\033[1m%s\033[0m", m.title)
	if m.filter == "" {
		b.WriteString("  \033[2mType to filter…\033[0m")
	} else {
		fmt.Fprintf(&b, "  \033[36m%s\033[0m", m.filter)
	}
	b.WriteString("\n\n")

	filtered := m.filteredItems()
	if len(filtered) == 0 {
		b.WriteString("  \033[2mNo matching models\033[0m\n")
	} else {
		end := m.offset + pageSize
		if end > len(filtered) {
			end = len(filtered)
		}
		if m.offset > 0 {
			fmt.Fprintf(&b, "  \033[2m↑ %d more\033[0m\n", m.offset)
		}
		for i := m.offset; i < end; i++ {
			item := filtered[i]
			if i == m.cursor {
				fmt.Fprintf(&b, "\033[1;7m  ▸ %-32s\033[0m", item.Name)
			} else {
				fmt.Fprintf(&b, "    %-32s", item.Name)
			}
			if item.Description != "" {
				fmt.Fprintf(&b, "  \033[2m%s\033[0m", item.Description)
			}
			b.WriteByte('\n')
		}
		if remaining := len(filtered) - end; remaining > 0 {
			fmt.Fprintf(&b, "  \033[2m↓ %d more\033[0m\n", remaining)
		}
	}
	b.WriteString("\n\033[2m↑/↓ navigate • pgup/pgdn jump • type to filter • enter select • esc cancel\033[0m")
	return b.String()
}

type ConfirmDefault int

const (
	ConfirmDefaultYes ConfirmDefault = iota
	ConfirmDefaultNo
)

type ConfirmOptions struct {
	YesLabel string
	NoLabel  string
	Default  ConfirmDefault
}

type confirmModel struct {
	prompt    string
	yesLabel  string
	noLabel   string
	yes       bool
	confirmed bool
	cancelled bool
}

func newConfirmModel(prompt string, options ConfirmOptions) confirmModel {
	yesLabel, noLabel := options.YesLabel, options.NoLabel
	if yesLabel == "" {
		yesLabel = "Yes"
	}
	if noLabel == "" {
		noLabel = "No"
	}
	return confirmModel{
		prompt: prompt, yesLabel: yesLabel, noLabel: noLabel,
		yes: options.Default != ConfirmDefaultNo,
	}
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		m.confirmed = true
		return m, tea.Quit
	case tea.KeyLeft, tea.KeyUp:
		m.yes = true
	case tea.KeyRight, tea.KeyDown, tea.KeyTab:
		m.yes = false
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			switch unicode.ToLower(key.Runes[0]) {
			case 'y':
				m.yes, m.confirmed = true, true
				return m, tea.Quit
			case 'n':
				m.yes, m.confirmed = false, true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.confirmed || m.cancelled {
		return ""
	}
	yes, no := " "+m.yesLabel+" ", " "+m.noLabel+" "
	if m.yes {
		yes = "\033[1;7m" + yes + "\033[0m"
		no = "\033[2m" + no + "\033[0m"
	} else {
		yes = "\033[2m" + yes + "\033[0m"
		no = "\033[1;7m" + no + "\033[0m"
	}
	return "\033[1m" + m.prompt + "\033[0m\n\n  " + yes + "  " + no +
		"\n\n\033[2m←/→ navigate • enter confirm • esc cancel\033[0m"
}

func Interactive(input io.Reader, output io.Writer) bool {
	if os.Getenv("LAUNCHPAD_ASSUME_TTY") == "1" {
		return true
	}
	in, inOK := input.(*os.File)
	out, outOK := output.(*os.File)
	return inOK && outOK && term.IsTerminal(int(in.Fd())) && term.IsTerminal(int(out.Fd()))
}

func Select(title string, items []Item, input io.Reader, output io.Writer) (string, error) {
	if !Interactive(input, output) {
		return "", errors.New("model selection requires an interactive terminal; use --model <model>")
	}
	result, err := tea.NewProgram(
		newSelectorModel(title, items),
		tea.WithInput(input),
		tea.WithOutput(output),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return "", err
	}
	selected, ok := result.(selectorModel)
	if !ok || selected.cancelled || selected.selected == "" {
		return "", ErrCancelled
	}
	return selected.selected, nil
}

func Confirm(prompt string, options ConfirmOptions, input io.Reader, output io.Writer) (bool, error) {
	if !Interactive(input, output) {
		return false, errors.New("confirmation requires an interactive terminal; re-run with --yes")
	}
	result, err := tea.NewProgram(
		newConfirmModel(prompt, options),
		tea.WithInput(input),
		tea.WithOutput(output),
	).Run()
	if err != nil {
		return false, err
	}
	confirmed, ok := result.(confirmModel)
	if !ok || confirmed.cancelled {
		return false, ErrCancelled
	}
	return confirmed.yes, nil
}

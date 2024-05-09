package label_map

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	lbl     string
	tooltip string
	input   textinput.Model
}

type ()

func New(label string, tooltip string) Model {
	return Model{
		lbl:     label,
		tooltip: tooltip,
		input:   textinput.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
¸

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	//switch msg := msg.(type) {
	//}
	return m, nil
}

func (m Model) View() string {
	return ""
}

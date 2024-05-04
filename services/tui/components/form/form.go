package form

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Control interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Control, tea.Cmd)
	View() string
	Blur()
	Focus() tea.Cmd
}

package msgs

import tea "github.com/charmbracelet/bubbletea"

type (
	ResMsg         = string
	ErrMsg         struct{ Err error }
	RouteChangeMsg struct {
		Route string
	}
)

func (e ErrMsg) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func RouteChangeCmd(route string) tea.Cmd {
	return func() tea.Msg {
		return RouteChangeMsg{route}
	}
}

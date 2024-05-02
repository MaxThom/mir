package msgs

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	ReqMsg string
	ResMsg = string
	ErrMsg struct {
		Err     error
		Timeout time.Duration
	}
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

func ReqMsgCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return ReqMsg(msg)
	}
}

func ResMsgCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return ResMsg(msg)
	}
}

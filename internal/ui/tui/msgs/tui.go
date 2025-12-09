package msgs

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	DefaultTimeout = 2 * time.Second
)

type (
	ReqMsg struct {
		Msg     string
		Timeout time.Duration
	}
	ResMsg struct {
		Msg     string
		Timeout time.Duration
	}
	ErrMsg struct {
		Err     error
		Timeout time.Duration
	}
	RouteChangeMsg struct {
		Route string
		Data  any
	}
	RouteResumeMsg struct {
		Route string
		Data  any
	}
)

func (e ErrMsg) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func ErrCmd(e error, t time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ErrMsg{e, t}
	}
}

func RouteChangeCmd(route string) tea.Cmd {
	return func() tea.Msg {
		return RouteChangeMsg{Route: route, Data: nil}
	}
}

func RouteChangeWithDataCmd(route string, data any) tea.Cmd {
	return func() tea.Msg {
		return RouteChangeMsg{Route: route, Data: data}
	}
}

func RouteResume(route string) tea.Cmd {
	return func() tea.Msg {
		return RouteResumeMsg{Route: route, Data: nil}
	}
}

func RouteResumeWithData(route string, data any) tea.Cmd {
	return func() tea.Msg {
		return RouteResumeMsg{Route: route, Data: data}
	}
}

func ReqMsgCmd(msg string, t time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ReqMsg{msg, t}
	}
}

func ResMsgCmd(msg string, t time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ResMsg{msg, t}
	}
}

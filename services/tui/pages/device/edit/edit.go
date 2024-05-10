package device_edit

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l = log.With().Str("page", "device_edit").Logger()
	v strings.Builder
)

type (
	DeviceFetchedMsg struct {
		devices []*core.Device
	}
)

type Model struct {
	ctx           context.Context
	help          mir_help.Model
	deviceIdInput textinput.Model
}

var styles = map[string]lipgloss.Style{
	"mir": lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")),
}

func NewModel(ctx context.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Blur()
	ti.CharLimit = 256
	ti.Width = 50
	ti.ShowSuggestions = true

	return &Model{
		ctx:           ctx,
		help:          mir_help.New(keys, []string{}, ""),
		deviceIdInput: ti,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(msgs.ReqMsgCmd("fetching devices..."))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case DeviceFetchedMsg:
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.devices)))
	case tea.KeyMsg:

		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")
	v.WriteString("" + m.deviceIdInput.View() + "\n")
	v.WriteString("\n\n" + m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["search"], k["create"], k["edit"]},
		{k["up"], k["down"]},
	}
}

var keys = keyMap{
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
}

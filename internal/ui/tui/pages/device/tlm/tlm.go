package device_telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l zerolog.Logger
	v strings.Builder
)

const ()

type Model struct {
	ctx  context.Context
	help mir_help.Model
}

type InputData struct {
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_tlm_list").Logger()

	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, "mir device telemetry"),
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	devs, ok := d.([]mir_v1.Device)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no device specified"), 2*time.Second),
			msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
		)
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		msgs.ErrCmd(fmt.Errorf("no device specified"), 2*time.Second),
		msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")

	v.WriteString(m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["search"], k["create"], k["edit"], k["delete"], k["schema"], k["tlm"], k["cmd"], k["cfg"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["search"], k["space"]},
		{k["create"], k["edit"], k["delete"]},
		{k["schema"], k["tlm"], k["cmd"], k["cfg"]},
		{k["up"], k["down"]},
	}
}

var keys = keyMap{
	"space": key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select/deselect"),
	),
	"search": key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	"create": key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create"),
	),
	"edit": key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	"delete": key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete"),
	),
	"schema": key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "schema"),
	),
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	"tlm": key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "telemetry"),
	),
	"cmd": key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "command"),
	),
	"cfg": key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "config"),
	),
}

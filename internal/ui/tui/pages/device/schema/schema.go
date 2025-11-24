package device_schema

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	l zerolog.Logger
	v strings.Builder
)

type Model struct {
	ctx  context.Context
	help mir_help.Model
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_schema").Logger()
	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, ""),
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	dev, ok := d.(mir_v1.Device)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no device specified"), 2*time.Second),
			msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
		)
	}
	sch, err := mir_proto.DecompressSchema(dev.Status.Schema.CompressedSchema)
	if err != nil {
		return tea.Batch(
			msgs.ErrCmd(err, 2*time.Second),
			msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
		)
	}
	rj, e := sch.ToYaml()
	if e != nil {
		l.Error().Err(e).Msg("")
		return tea.Batch(
			msgs.ErrCmd(e, 2*time.Second),
			msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
		)
	}

	headerComments := []string{
		"Explore the protbuf schema of the device below",
		"DeviceId: " + dev.Spec.DeviceId,
	}
	return msgs.OpenEditorCmd(msgs.FileTypeYAML, rj, headerComments)
}

func (m *Model) Init() tea.Cmd {
	var rj []byte
	headerComments := []string{
		"Explore the protbuf schema of the device below",
	}
	return msgs.OpenEditorCmd(msgs.FileTypeYAML, rj, headerComments)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.EditorFinishedMsg:
		var cmds []tea.Cmd
		if msg.Err != nil {
			cmds = append(cmds, msgs.ErrCmd(msg.Err, 2*time.Second))
			cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
		} else {
			cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
		}
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
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

package device_edit

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
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// IDEA wide option that show more fields

var (
	l zerolog.Logger
	v strings.Builder
)

type Model struct {
	ctx  context.Context
	help mir_help.Model
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_edit").Logger()
	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, ""),
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	rj, e := yaml.Marshal(d)
	if e != nil {
		l.Error().Err(e).Msg("")
		return tea.Batch(msgs.ErrCmd(e, 2*time.Second), msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
	}
	l.Debug().Str("edit", "before").Msg(fmt.Sprintf("%v", string(rj)))

	headerComments := []string{
		"Edit the device below",
		"To remove a field, you must explicitly set it to null",
		"Only fields under meta and properties.desired are editable",
	}
	return msgs.OpenEditorCmd(msgs.FileTypeYAML, rj, headerComments)
}

func (m *Model) Init() tea.Cmd {
	var rj []byte
	headerComments := []string{
		"Edit the device below",
		"To remove a field, you must explicitly set it to null",
		"Only fields under meta and properties.desired are editable",
	}
	return msgs.OpenEditorCmd(msgs.FileTypeYAML, rj, headerComments)
}

func (m Model) Resume() tea.Cmd {
	return nil
}

func (m Model) ResumeWithData(d any) tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceUpdateMsg:
		rsp := "device edited successfully"
		if len(msg.Devices) > 0 {
			rsp = fmt.Sprintf("device '%s' edited successfully", msg.Devices[0].Spec.DeviceId)
		}
		return m, tea.Batch(msgs.ResMsgCmd(rsp, msgs.DefaultTimeout), msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
	case msgs.EditorFinishedMsg:
		var cmds []tea.Cmd
		if msg.Err != nil {
			l.Debug().Err(msg.Err).Msg("")
			cmds = append(cmds, msgs.ErrCmd(msg.Err, 2*time.Second))
			cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
		} else {
			l.Debug().Str("edit", "after").Msg(fmt.Sprintf("%v", string(msg.Content)))
			dev := mir_v1.Device{}
			if err := yaml.Unmarshal(msg.Content, &dev); err != nil {
				l.Error().Err(err).Msg("can't unmarshal edited device")
				cmds = append(cmds, msgs.ErrCmd(err, 2*time.Second))
				cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
			} else {
				l.Debug().Str("edit", "unmarshalled").Msg(fmt.Sprintf("%v", dev))
				cmds = append(cmds, msgs.UpdateMirDevice(store.Bus, dev))
			}
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

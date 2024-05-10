package device_edit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/msgs"
	device_list "github.com/maxthom/mir/services/tui/pages/device/list"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l zerolog.Logger
	v strings.Builder
)

type (
	DeviceFetchedMsg struct {
		devices []*core.Device
	}
	EditorFinishedMsg struct {
		content json.RawMessage
		err     error
	}
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
	l.Debug().Msg(fmt.Sprintf("%v", d))
	rj, e := json.MarshalIndent(d, "", "  ")
	if e != nil {
		l.Error().Err(e).Msg("")
		return tea.Batch(msgs.ErrCmd(e, 2*time.Second), msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))

	}
	return openEditor(rj)
}

func (m *Model) Init() tea.Cmd {
	var rj json.RawMessage
	return openEditor(rj)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case DeviceFetchedMsg:
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.devices)))
	case EditorFinishedMsg:
		var res tea.Cmd
		if msg.err != nil {
			res = msgs.ErrCmd(msg.err, 2*time.Second)
		} else {
			// TODO update device
			res = msgs.ResMsgCmd("device edited successfully")
		}

		return m, tea.Batch(res, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
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

func openEditor(data json.RawMessage) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Create a temporary file
	// TODO option for user to switch between json and yaml
	tmpfile, err := os.CreateTemp("", "twin_*.json")
	if err != nil {
		l.Error().Err(err).Msg("can't create temporary file for editing twin")
	}

	// Write initial device to the temp file
	_, err = tmpfile.Write(data)
	if err != nil {
		l.Error().Err(err).Msg("can't write to temporary file for editing twin")
	}
	tmpfile.Close() // Close the file as the editor will need to open it

	// Open the editor
	c := exec.Command(editor, tmpfile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			l.Error().Err(err).Msg("can't run editor command for editing twin")
			return EditorFinishedMsg{content: []byte{}, err: err}
		}
		// Read the modified file
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			l.Error().Err(err).Msg("can't write to temporary file for editing twin")
			return EditorFinishedMsg{content: []byte{}, err: err}
		}

		err = os.Remove(tmpfile.Name())
		if err != nil {
			l.Error().Err(err).Msg("can't delete temporary file for editing twin")
			return EditorFinishedMsg{content: content, err: err}
		}

		return EditorFinishedMsg{content: content, err: nil}
	})
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

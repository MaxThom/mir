package device_edit

import (
	"context"
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
	EditorFinishedMsg struct {
		content []byte
		err     error
	}
)

type Model struct {
	ctx     context.Context
	help    mir_help.Model
	content string
}

func NewModel(ctx context.Context) *Model {

	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, ""),
	}
}

func (m *Model) Init() tea.Cmd {
	return openEditor() //tea.Batch(msgs.ReqMsgCmd("fetching devices..."))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case DeviceFetchedMsg:
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.devices)))
	case EditorFinishedMsg:
		var res tea.Cmd
		if msg.err != nil {
			m.content = string(msg.content) + "\n" + msg.err.Error()
			res = msgs.ErrCmd(msg.err, 2*time.Second)
		} else {
			m.content = string(msg.content)
			res = msgs.ResMsgCmd("device edited successfully")
		}

		return m, tea.Batch(res, msgs.RouteChangeCmd("/devices"))
	case tea.KeyMsg:

		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString(m.content)
	v.WriteString("\n")
	v.WriteString("\n\n" + m.help.View())
	return v.String()
}

func openEditor() tea.Cmd {
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
	_, err = tmpfile.Write([]byte(`{"key": "value"}`))
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

		l.Error().Err(err).Msg("can't run editor command for editing twin")

		// Read the modified file
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			l.Error().Err(err).Msg("can't write to temporary file for editing twin")
		}

		os.Remove(tmpfile.Name())
		return EditorFinishedMsg{content: content, err: err}
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

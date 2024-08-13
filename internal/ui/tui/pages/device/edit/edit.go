package device_edit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l zerolog.Logger
	v strings.Builder
)

type (
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
	rj, e := json.MarshalIndent(d, "", "  ")
	if e != nil {
		l.Error().Err(e).Msg("")
		return tea.Batch(msgs.ErrCmd(e, 2*time.Second), msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))

	}
	l.Debug().Str("edit", "before").Msg(fmt.Sprintf("%v", string(rj)))
	return openEditor(rj)
}

func (m *Model) Init() tea.Cmd {
	var rj json.RawMessage
	return openEditor(rj)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceUpdateMsg:
		rsp := "device edited successfully"
		if len(msg.Devices) > 0 {
			rsp = fmt.Sprintf("device '%s' edited successfully", msg.Devices[0].Spec.DeviceId)
		}
		return m, tea.Batch(msgs.ResMsgCmd(rsp), msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
	case EditorFinishedMsg:
		var cmds []tea.Cmd
		if msg.err != nil {
			cmds = append(cmds, msgs.ErrCmd(msg.err, 2*time.Second))
			cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
		} else {
			l.Debug().Str("edit", "after").Msg(fmt.Sprintf("%v", string(msg.content)))
			dev := mir_models.Device{}
			if err := json.Unmarshal(msg.content, &dev); err != nil {
				l.Error().Err(err).Msg("can't unmarshal edited device")
				cmds = append(cmds, msgs.ErrCmd(err, 2*time.Second))
				cmds = append(cmds, msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}))
			} else {
				l.Debug().Str("edit", "unmarshalled").Msg(fmt.Sprintf("%v", dev))
				cmds = append(cmds, msgs.UpdateMirDevice(store.Bus, mir_models.NewUpdateDeviceMetaReqFromDevice(dev)))
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
	var comments strings.Builder
	comments.WriteString("// Edit the device below\n")
	comments.WriteString("// To remove a field, you must explicitly set it to null\n")
	comments.WriteString("// Only fields under meta and properties.desired are editable\n")
	_, err = tmpfile.Write([]byte(comments.String()))
	if err != nil {
		l.Error().Err(err).Msg("can't write to temporary file for editing twin")
	}
	_, err = tmpfile.Write(data)
	if err != nil {
		l.Error().Err(err).Msg("can't write to temporary file for editing twin")
	}
	// Close the file as the editor will need to open it
	tmpfile.Close()

	c := exec.Command(editor, tmpfile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			l.Error().Err(err).Msg("can't run editor command for editing twin")
			return EditorFinishedMsg{content: []byte{}, err: err}
		}

		file, err := os.Open(tmpfile.Name())
		if err != nil {
			l.Error().Err(err).Msg("can't open to temporary file for reading twin")
			return EditorFinishedMsg{content: []byte{}, err: err}
		}

		var sb strings.Builder
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				sb.WriteString(line)
			}
		}
		content := []byte(sb.String())

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

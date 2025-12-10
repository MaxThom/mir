package device_configuration_current

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/components/menu"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/internal/ui/tui/styles"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	l zerolog.Logger
	v strings.Builder
)

const ()

type Model struct {
	ctx     context.Context
	help    mir_help.Model
	cfgReq  *mir_apiv1.SendConfigRequest
	cfgResp map[string]*mir_apiv1.SendConfigResponse_ConfigResponse
	vp      viewport.Model
	list    menu.Model
}

type InputData struct {
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_cfg_resp").Logger()

	vp := viewport.New(store.ScreenWidth, store.ScreenHeight)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, "mir config responses"),
		vp:   vp,
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	m.vp.Height = store.ScreenHeight - 5
	m.vp.Width = 75
	req, ok := d.(*mir_apiv1.SendConfigRequest)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no config specified"), 2*time.Second),
			msgs.RouteResume("/devices/configs"),
		)
	}
	m.cfgReq = req

	return tea.Batch(
		msgs.ReqMsgCmd("Config '"+req.Name+"' sent to "+strconv.Itoa(len(req.Targets.Ids))+" devices", msgs.DefaultTimeout),
		msgs.SendMirDeviceConfigs(store.Bus, req),
	)
}

func (m Model) Resume() tea.Cmd {
	return nil
}

func (m Model) ResumeWithData(d any) tea.Cmd {
	return nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		msgs.ErrCmd(fmt.Errorf("no configs specified"), 2*time.Second),
		msgs.RouteResume("/devices/configs"),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdVp tea.Cmd
	var cmdKey tea.Cmd
	var cmdList tea.Cmd
	var cmdRes tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp.Width = msg.Width - 4
		m.vp.Height = msg.Height - 8
	case msgs.DeviceConfigSentMsg:
		m.cfgResp = msg.ConfigsResponses
		m.list = menu.New(m.renderCmdResp(m.cfgResp))
		cmdRes = msgs.ResMsgCmd(strconv.Itoa(len(msg.ConfigsResponses))+" config responses received", 5*time.Second)
	case tea.KeyMsg:
		m.help, cmdKey = m.help.Update(msg)
		m.list, cmdList = m.list.Update(msg)
		if cmdList != nil {
			c := cmdList()
			mv, ok := c.(menu.CursorMovedMsg)
			if ok {
				lineCount := len(strings.Split(m.list.GetChoice().Description, "\n"))
				if mv.Position == 0 {
					m.vp.GotoTop()
				} else if len(m.cfgResp)-1 == mv.Position {
					m.vp.GotoBottom()
				} else {
					m.vp.ScrollDown(mv.Direction * lineCount)
				}
			}
		}
		if msg.String() == "r" {
			return m, m.InitWithData(m.cfgReq)
		} else if msg.String() == "e" {
			errorIds := []string{}
			for _, resp := range m.cfgResp {
				if resp.Status == mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR {
					errorIds = append(errorIds, resp.DeviceId)
				}
			}
			if len(errorIds) > 0 {
				m.cfgReq.Targets.Ids = errorIds
				return m, m.InitWithData(m.cfgReq)
			} else {
				return m, msgs.ResMsgCmd("No config in error", 5*time.Second)
			}
		}
	}

	m.vp, cmdVp = m.vp.Update(msg)
	return m, tea.Batch(cmdVp, cmdRes, cmdKey, cmdList)
}

func (m *Model) View() string {
	v.Reset()
	header := styles.Help.Bold(false).Render(fmt.Sprintf("Configuration responses for %d devices\n", len(m.cfgResp)))
	m.vp.SetContent(header + m.list.View())
	v.WriteString(m.vp.View())
	v.WriteString(m.help.View())
	return v.String()
}

func (m *Model) renderCmdResp(resps map[string]*mir_apiv1.SendConfigResponse_ConfigResponse) []menu.Option {
	i := 1
	choices := []menu.Option{}
	for k, v := range resps {
		var sb strings.Builder
		if v.Error != "" {
			errorText := v.Error
			if len(errorText) > 50 {
				lines := []string{}
				start := 0
				for start < len(errorText) {
					end := start + 50
					if end > len(errorText) {
						end = len(errorText)
					} else {
						// Find the next space after position 30
						spaceIdx := strings.IndexByte(errorText[end:], ' ')
						if spaceIdx != -1 {
							end += spaceIdx
						}
					}
					lines = append(lines, errorText[start:end])
					start = end
					// Skip the space if we broke at one
					if start < len(errorText) && errorText[start] == ' ' {
						start++
					}
				}
				sb.WriteString(strings.Join(lines, "\n    "))
			} else {
				sb.WriteString(errorText + "\n")
			}
		} else if len(v.Payload) > 0 {
			sb.WriteString(v.Name)
			sb.WriteString("\n")

			p, err := prettyPrintJSON(string(v.Payload))
			if err != nil {
				sb.WriteString(errors.Wrap(err, "    error unmarshaling JSON in terminal").Error())
			} else {
				sb.WriteString("    " + p)
			}
		}
		sb.WriteString("\n")

		lbl := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d. %s", i, k))
		st := ""
		switch v.Status {
		case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS:
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("SUCCESS")
		case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR:
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render("ERROR")
		case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_PENDING:
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("PENDING")
		case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED:
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("VALIDATED")
		case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_UNSPECIFIED:
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("UNSPECIFIED")
		}

		choices = append(choices, menu.Option{
			Value:       k,
			Label:       lbl + " " + st,
			Description: sb.String(),
		})
		i++
	}
	return choices
}

func prettyPrintJSON(jsonStr string) (string, error) {
	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}

	prettyJSON, err := json.MarshalIndent(obj, "    ", "  ")
	if err != nil {
		return "", err
	}

	return string(prettyJSON), nil
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["again"], k["again_error"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["again"], k["again_error"]},
	}
}

var keys = keyMap{
	"again": key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "resend command"),
	),
	"again_error": key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "resend command in error"),
	),
}

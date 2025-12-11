package device_configuration_response

import (
	"context"
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
	"github.com/maxthom/mir/internal/ui/tui/utils"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	l zerolog.Logger
	v strings.Builder
)

type Model struct {
	ctx      context.Context
	help     mir_help.Model
	cfgReq   *mir_apiv1.SendConfigRequest
	cfgResp  map[string]*mir_apiv1.SendConfigResponse_ConfigResponse
	fetching bool
	vp       viewport.Model
	list     menu.Model
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
	dims := utils.DefaultViewportDimensions()
	utils.UpdateViewportSize(&m.vp, 75, store.ScreenHeight-5, dims)
	req, ok := d.(*mir_apiv1.SendConfigRequest)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no config specified"), 2*time.Second),
			msgs.RouteResume("/devices/configs"),
		)
	}
	m.cfgReq = req
	m.fetching = true

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
		dims := utils.DefaultViewportDimensions()
		utils.UpdateViewportSize(&m.vp, msg.Width, msg.Height, dims)
	case msgs.DeviceConfigSentMsg:
		m.cfgResp = msg.ConfigsResponses
		m.list = menu.New(m.renderCmdResp(m.cfgResp))
		m.fetching = false
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
	header := ""
	if m.fetching {
		header = styles.Help.Bold(false).Render(fmt.Sprintf("Sending config '%s'...\n", m.cfgReq.Name))
	} else {
		header = styles.Help.Bold(false).Render(fmt.Sprintf("Configuration responses for %d devices\n", len(m.cfgResp)))
	}
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
			if len(errorText) > utils.DefaultWrapWidth {
				sb.WriteString(utils.WrapText(errorText, utils.DefaultWrapOptions()))
			} else {
				sb.WriteString(errorText + "\n")
			}
		} else if len(v.Payload) > 0 {
			sb.WriteString(v.Name)
			sb.WriteString("\n")

			p, err := utils.FormatJSON(string(v.Payload), "    ", utils.DefaultJSONIndent)
			if err != nil {
				sb.WriteString(errors.Wrap(err, "    error unmarshaling JSON in terminal").Error())
			} else {
				sb.WriteString("    " + p)
			}
		}
		sb.WriteString("\n")

		lbl := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d. %s", i, k))
		st := GetConfigStatusBadge(v.Status)

		choices = append(choices, menu.Option{
			Value:       k,
			Label:       lbl + " " + st,
			Description: sb.String(),
		})
		i++
	}
	return choices
}

// GetConfigStatusBadge returns a styled status badge for config responses
func GetConfigStatusBadge(status mir_apiv1.ConfigResponseStatus) string {
	switch status {
	case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS:
		return styles.RenderStatusBadge("SUCCESS", styles.StatusColors.Success)
	case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR:
		return styles.RenderStatusBadge("ERROR", styles.StatusColors.Error)
	case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_PENDING:
		return styles.RenderStatusBadge("PENDING", styles.StatusColors.Pending)
	case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED:
		return styles.RenderStatusBadge("VALIDATED", styles.StatusColors.Validated)
	case mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_UNSPECIFIED:
		return styles.RenderStatusBadge("UNSPECIFIED", styles.StatusColors.Warning)
	default:
		return styles.RenderStatusBadge("UNKNOWN", styles.StatusColors.Warning)
	}
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

package device_configuration_values

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
	ctx     context.Context
	help    mir_help.Model
	devCfgs []*mir_apiv1.DevicesConfigs_ConfigValues
	cfgName string
	targets *mir_apiv1.DeviceTarget
	vp      viewport.Model
	list    menu.Model
}

type InputData struct {
	CfgName string
	Targets *mir_apiv1.DeviceTarget
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
	req, ok := d.(*InputData)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no config specified"), 2*time.Second),
			msgs.RouteResume("/devices/configs"),
		)
	}
	m.cfgName = req.CfgName
	m.targets = req.Targets

	return tea.Batch(
		msgs.ReqMsgCmd("Config '"+req.CfgName+"' sent to "+strconv.Itoa(len(req.Targets.Ids))+" devices", msgs.DefaultTimeout),
		msgs.ListMirDeviceConfigs(store.Bus, m.targets),
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
	case msgs.DeviceConfigListedMsg:
		m.devCfgs = []*mir_apiv1.DevicesConfigs_ConfigValues{}
		for _, cfg := range msg.Configs {
			m.devCfgs = append(m.devCfgs, cfg.CfgValues...)
		}
		m.list = menu.New(m.renderCfgValues(m.cfgName, m.devCfgs))
		cmdRes = msgs.ResMsgCmd(fmt.Sprintf("%d configs fetched on %d devices", len(msg.Configs), len(m.targets.Ids)), msgs.DefaultTimeout)
	case tea.WindowSizeMsg:
		dims := utils.DefaultViewportDimensions()
		utils.UpdateViewportSize(&m.vp, msg.Width, msg.Height, dims)
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
				} else if len(m.devCfgs)-1 == mv.Position {
					m.vp.GotoBottom()
				} else {
					m.vp.ScrollDown(mv.Direction * lineCount)
				}
			}
		}
	}

	m.vp, cmdVp = m.vp.Update(msg)
	return m, tea.Batch(cmdVp, cmdKey, cmdRes, cmdList)
}

func (m *Model) View() string {
	v.Reset()
	header := styles.Help.Bold(false).Render(fmt.Sprintf("Configuration values for %d devices\n", len(m.targets.Ids)))
	m.vp.SetContent(header + m.list.View())
	v.WriteString(m.vp.View())
	v.WriteString(m.help.View())
	return v.String()
}

func (m *Model) renderCfgValues(cfgName string, cfgValues []*mir_apiv1.DevicesConfigs_ConfigValues) []menu.Option {
	choices := []menu.Option{}
	for i, v := range cfgValues {
		nameNs := v.Id.Name + "/" + v.Id.Namespace
		if nameNs == "/" {
			nameNs = v.Id.DeviceId
		}
		lbl := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d. %s", i+1, nameNs))

		st := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("SUCCESS")
		if v.Error != "" || len(v.Values) == 0 || cfgName == "" {
			st = lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render("ERROR")
		}

		var sb strings.Builder
		if v.Error != "" || len(v.Values) == 0 || cfgName == "" {
			errorText := v.Error
			if len(v.Values) == 0 || cfgName == "" {
				errorText = fmt.Errorf("cannot find properties for '%s'. Please refresh page.", cfgName).Error()
			}
			if len(errorText) > utils.DefaultWrapWidth {
				sb.WriteString(utils.WrapText(errorText, utils.DefaultWrapOptions()))
			} else {
				sb.WriteString(errorText + "\n")
			}
		} else {
			val := v.Values[cfgName]
			sb.WriteString(cfgName)
			sb.WriteString("\n")

			p, err := utils.FormatJSON(string(val), "    ", utils.DefaultJSONIndent)
			if err != nil {
				sb.WriteString(errors.Wrap(err, "    error unmarshaling JSON in terminal").Error())
				st = lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render("ERROR")
			} else {
				sb.WriteString("    " + p)
			}
		}
		sb.WriteString("\n")

		choices = append(choices, menu.Option{
			Value:       nameNs,
			Label:       lbl + " " + st,
			Description: sb.String(),
		})
	}
	return choices
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

var keys = keyMap{}

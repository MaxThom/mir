package msgs

import (
	tea "github.com/charmbracelet/bubbletea"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

type (
	DeviceCommandListedMsg struct {
		Commands []*mir_apiv1.DevicesCommands
	}
)

func ListMirDeviceCommands(m *mir.Mir, t *mir_apiv1.DeviceTarget) tea.Cmd {
	return listMirDeviceCommandsCmd(m, t)
}

func listMirDeviceCommandsCmd(m *mir.Mir, t *mir_apiv1.DeviceTarget) func() tea.Msg {
	return func() tea.Msg {
		list, err := m.Client().ListCommands().Request(&mir_apiv1.SendListCommandsRequest{Targets: t})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceCommandListedMsg{Commands: list}
	}
}

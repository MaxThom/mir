package msgs

import (
	tea "github.com/charmbracelet/bubbletea"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

type (
	DeviceTelemetryListedMsg struct {
		Telemetry []*mir_apiv1.DevicesTelemetry
	}
)

func ListMirDeviceTelemetry(m *mir.Mir, t *mir_apiv1.DeviceTarget) tea.Cmd {
	return listMirDeviceTelemetryCmd(m, t)
}

func listMirDeviceTelemetryCmd(m *mir.Mir, t *mir_apiv1.DeviceTarget) func() tea.Msg {
	return func() tea.Msg {
		list, err := m.Client().ListTelemetry().Request(&mir_apiv1.SendListTelemetryRequest{Targets: t})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceTelemetryListedMsg{Telemetry: list}
	}
}

package msgs

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

type (
	DeviceCommandListedMsg struct {
		Commands []*mir_apiv1.DevicesCommands
	}
	DeviceCommandSentMsg struct {
		CommandsResponse map[string]*mir_apiv1.SendCommandResponse_CommandResponse
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

func SendMirDeviceCommands(m *mir.Mir, req *mir_apiv1.SendCommandRequest) tea.Cmd {
	return sendMirDeviceCommandsCmd(m, req)
}

func SendMirDeviceCommandsRaw(m *mir.Mir, t *mir_apiv1.DeviceTarget, cmd string, payload json.RawMessage) tea.Cmd {
	req := mir_apiv1.SendCommandRequest{
		Name:            cmd,
		Payload:         payload,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Targets:         t,
	}
	return sendMirDeviceCommandsCmd(m, &req)
}

func sendMirDeviceCommandsCmd(m *mir.Mir, req *mir_apiv1.SendCommandRequest) func() tea.Msg {
	return func() tea.Msg {
		cmdResp, err := m.Client().SendCommand().Request(req)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceCommandSentMsg{CommandsResponse: cmdResp}
	}
}

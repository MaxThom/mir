package msgs

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

type (
	DeviceConfigListedMsg struct {
		Configs []*mir_apiv1.DevicesConfigs
	}
	DeviceConfigSentMsg struct {
		ConfigsResponses map[string]*mir_apiv1.SendConfigResponse_ConfigResponse
	}
)

func ListMirDeviceConfigs(m *mir.Mir, t *mir_apiv1.DeviceTarget) tea.Cmd {
	return listMirDeviceConfigsCmd(m, t)
}

func listMirDeviceConfigsCmd(m *mir.Mir, t *mir_apiv1.DeviceTarget) func() tea.Msg {
	return func() tea.Msg {
		list, err := m.Client().ListConfig().Request(&mir_apiv1.SendListConfigRequest{Targets: t})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceConfigListedMsg{Configs: list}
	}
}

func SendMirDeviceConfigs(m *mir.Mir, req *mir_apiv1.SendConfigRequest) tea.Cmd {
	return sendMirDeviceConfigsCmd(m, req)
}

func SendMirDeviceConfigsRaw(m *mir.Mir, t *mir_apiv1.DeviceTarget, cmd string, payload json.RawMessage) tea.Cmd {
	req := mir_apiv1.SendConfigRequest{
		Name:            cmd,
		Payload:         payload,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Targets:         t,
	}
	return sendMirDeviceConfigsCmd(m, &req)
}

func sendMirDeviceConfigsCmd(m *mir.Mir, req *mir_apiv1.SendConfigRequest) func() tea.Msg {
	return func() tea.Msg {
		cfgResp, err := m.Client().SendConfig().Request(req)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceConfigSentMsg{ConfigsResponses: cfgResp}
	}
}

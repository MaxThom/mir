package msgs

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

type (
	DeviceListedMsg struct {
		Devices []mir_v1.Device
		NoToast bool
	}
	DeviceCreatedMsg struct {
		Devices []mir_v1.Device
		NoToast bool
	}
	DeviceDeleteMsg struct {
		Devices []mir_v1.Device
		NoToast bool
	}
	DeviceUpdateMsg struct {
		Devices []mir_v1.Device
		NoToast bool
	}
)

func ListMirDevicesSilently(m *mir.Mir) tea.Cmd {
	return listMirDevicesCmd(m, true)
}

func ListMirDevices(m *mir.Mir) tea.Cmd {
	return listMirDevicesCmd(m, false)
}

func listMirDevicesCmd(m *mir.Mir, noToast bool) func() tea.Msg {
	return func() tea.Msg {
		list, err := m.Client().ListDevice().Request(mir_v1.DeviceTarget{}, false)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceListedMsg{Devices: list, NoToast: noToast}
	}
}

func CreateMirDeviceSilently(m *mir.Mir, d mir_v1.Device) tea.Cmd {
	return createMirDeviceCmd(m, true, d)
}

func CreateMirDevice(m *mir.Mir, d mir_v1.Device) tea.Cmd {
	return createMirDeviceCmd(m, false, d)
}

func createMirDeviceCmd(m *mir.Mir, noToast bool, d mir_v1.Device) tea.Cmd {
	return func() tea.Msg {
		dev, err := m.Client().CreateDevice().Request(d)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceCreatedMsg{Devices: []mir_v1.Device{dev}, NoToast: noToast}
	}
}

func DeleteMirDeviceSilently(m *mir.Mir, t mir_v1.DeviceTarget) tea.Cmd {
	return deleteMirDeviceCmd(m, true, t)
}

func DeleteMirDevice(m *mir.Mir, t mir_v1.DeviceTarget) tea.Cmd {
	return deleteMirDeviceCmd(m, false, t)
}

func deleteMirDeviceCmd(m *mir.Mir, noToast bool, t mir_v1.DeviceTarget) tea.Cmd {
	return func() tea.Msg {
		devs, err := m.Client().DeleteDevice().Request(t)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceDeleteMsg{Devices: devs, NoToast: noToast}
	}
}

func UpdateMirDeviceSilently(m *mir.Mir, d mir_v1.Device) tea.Cmd {
	return updateMirDeviceCmd(m, true, d)
}

func UpdateMirDevice(m *mir.Mir, d mir_v1.Device) tea.Cmd {
	return updateMirDeviceCmd(m, false, d)
}

func updateMirDeviceCmd(m *mir.Mir, noToast bool, d mir_v1.Device) tea.Cmd {
	return func() tea.Msg {
		devs, err := m.Client().UpdateDevice().RequestSingle(d)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return DeviceUpdateMsg{Devices: devs, NoToast: noToast}
	}
}

package msgs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
)

type (
	DeviceListedMsg struct {
		Devices []*core_apiv1.Device
		NoToast bool
	}
	DeviceCreatedMsg struct {
		Devices []*core_apiv1.Device
		NoToast bool
	}
	DeviceDeleteMsg struct {
		Devices []*core_apiv1.Device
		NoToast bool
	}
	DeviceUpdateMsg struct {
		Devices []*core_apiv1.Device
		NoToast bool
	}
)

func ListMirDevicesSilently(bus *bus.BusConn) tea.Cmd {
	return listMirDevicesCmd(bus, true)
}

func ListMirDevices(bus *bus.BusConn) tea.Cmd {
	return listMirDevicesCmd(bus, false)
}

func listMirDevicesCmd(bus *bus.BusConn, noToast bool) func() tea.Msg {
	return func() tea.Msg {
		resp, err := core_client.PublishDeviceListRequest(bus, &core_apiv1.ListDeviceRequest{
			Targets: &core_apiv1.Targets{},
		})
		if err != nil {
			// TODO move error from cli to next to core client and use it here as well
			//l.Error().Err(err).Msg("")
			return ErrMsg{Err: err}
		}
		if resp.GetError() != nil {
			return ErrMsg{Err: fmt.Errorf(resp.GetError().GetMessage())}
		}
		return DeviceListedMsg{Devices: resp.GetOk().Devices, NoToast: noToast}
	}
}

func CreateMirDeviceSilently(bus *bus.BusConn, req *core_apiv1.CreateDeviceRequest) tea.Cmd {
	return createMirDeviceCmd(bus, true, req)
}

func CreateMirDevice(bus *bus.BusConn, req *core_apiv1.CreateDeviceRequest) tea.Cmd {
	return createMirDeviceCmd(bus, false, req)
}

func createMirDeviceCmd(bus *bus.BusConn, noToast bool, req *core_apiv1.CreateDeviceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := core_client.PublishDeviceCreateRequest(bus, req)
		if err != nil {
			// TODO move error from cli to next to core client and use it here as well
			//l.Error().Err(err).Msg("")
			return ErrMsg{Err: err}
		}
		if resp.GetError() != nil {
			return ErrMsg{Err: fmt.Errorf(resp.GetError().GetMessage())}
		}
		return DeviceCreatedMsg{Devices: resp.GetOk().Devices, NoToast: noToast}
	}
}

func DeleteMirDeviceSilently(bus *bus.BusConn, req *core_apiv1.DeleteDeviceRequest) tea.Cmd {
	return deleteMirDeviceCmd(bus, true, req)
}

func DeleteMirDevice(bus *bus.BusConn, req *core_apiv1.DeleteDeviceRequest) tea.Cmd {
	return deleteMirDeviceCmd(bus, false, req)
}

func deleteMirDeviceCmd(bus *bus.BusConn, noToast bool, req *core_apiv1.DeleteDeviceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := core_client.PublishDeviceDeleteRequest(bus, req)
		if err != nil {
			// TODO move error from cli to next to core client and use it here as well
			//l.Error().Err(err).Msg("")
			return ErrMsg{Err: err}
		}
		if resp.GetError() != nil {
			return ErrMsg{Err: fmt.Errorf(resp.GetError().GetMessage())}
		}
		return DeviceDeleteMsg{Devices: resp.GetOk().Devices, NoToast: noToast}
	}
}

func UpdateMirDeviceSilently(bus *bus.BusConn, req *core_apiv1.UpdateDeviceRequest) tea.Cmd {
	return updateMirDeviceCmd(bus, true, req)
}

func UpdateMirDevice(bus *bus.BusConn, req *core_apiv1.UpdateDeviceRequest) tea.Cmd {
	return updateMirDeviceCmd(bus, false, req)
}

func updateMirDeviceCmd(bus *bus.BusConn, noToast bool, req *core_apiv1.UpdateDeviceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := core_client.PublishDeviceUpdateRequest(bus, req)
		if err != nil {
			// TODO move error from cli to next to core client and use it here as well
			//l.Error().Err(err).Msg("")
			return ErrMsg{Err: err}
		}
		if resp.GetError() != nil {
			return ErrMsg{Err: fmt.Errorf(resp.GetError().GetMessage())}
		}
		return DeviceUpdateMsg{Devices: resp.GetOk().Devices, NoToast: noToast}
	}
}

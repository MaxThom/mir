package msgs

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	client_core "github.com/maxthom/mir/services/core"
	"github.com/rs/zerolog/log"
)

var (
	l = log.With().Str("store", "msgs").Logger()
)

type (
	ReqMsg string
	ResMsg = string
	ErrMsg struct {
		Err     error
		Timeout time.Duration
	}
	RouteChangeMsg struct {
		Route string
	}
	DeviceListedMsg struct {
		Devices []*core.Device
		NoToast bool
	}
	DeviceCreatedMsg struct {
		Devices []*core.Device
		NoToast bool
	}
	DeviceDeleteMsg struct {
		Devices []*core.Device
		NoToast bool
	}
)

func (e ErrMsg) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func ErrCmd(e error, t time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ErrMsg{e, t}
	}
}

func RouteChangeCmd(route string) tea.Cmd {
	return func() tea.Msg {
		return RouteChangeMsg{route}
	}
}

func ReqMsgCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return ReqMsg(msg)
	}
}

func ResMsgCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return ResMsg(msg)
	}
}

func ListMirDevicesSilently(bus *bus.BusConn) tea.Cmd {
	return listMirDevicesCmd(bus, true)
}

func ListMirDevices(bus *bus.BusConn) tea.Cmd {
	return listMirDevicesCmd(bus, false)
}

func listMirDevicesCmd(bus *bus.BusConn, noToast bool) func() tea.Msg {
	return func() tea.Msg {
		resp, err := client_core.PublishDeviceListRequest(bus, &core.ListDeviceRequest{
			Targets: &core.Targets{},
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

func CreateMirDeviceSilently(bus *bus.BusConn, req *core.CreateDeviceRequest) tea.Cmd {
	return createMirDeviceCmd(bus, true, req)
}

func CreateMirDevice(bus *bus.BusConn, req *core.CreateDeviceRequest) tea.Cmd {
	return createMirDeviceCmd(bus, false, req)
}

func createMirDeviceCmd(bus *bus.BusConn, noToast bool, req *core.CreateDeviceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := client_core.PublishDeviceCreateRequest(bus, req)
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

func DeleteMirDeviceSilently(bus *bus.BusConn, req *core.DeleteDeviceRequest) tea.Cmd {
	return deleteMirDeviceCmd(bus, true, req)
}

func DeleteMirDevice(bus *bus.BusConn, req *core.DeleteDeviceRequest) tea.Cmd {
	return deleteMirDeviceCmd(bus, false, req)
}

func deleteMirDeviceCmd(bus *bus.BusConn, noToast bool, req *core.DeleteDeviceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := client_core.PublishDeviceDeleteRequest(bus, req)
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

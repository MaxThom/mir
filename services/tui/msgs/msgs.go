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
	DeviceFetchedMsg struct {
		Devices []*core.Device
	}
	DeviceCreatedMsg struct {
		Devices []*core.Device
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

func FetchMirDevices(bus *bus.BusConn) tea.Cmd {
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
		return DeviceFetchedMsg{Devices: resp.GetOk().Devices}
	}
}

func CreateMirDevice(bus *bus.BusConn, req *core.CreateDeviceRequest) tea.Cmd {
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
		return DeviceCreatedMsg{Devices: resp.GetOk().Devices}
	}
}

package cli

import (
	"fmt"
	"strings"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
)

type MirHttpError struct {
	Code    uint32
	Message string
	Details []string
}

func (e MirHttpError) Error() string {
	return fmt.Sprintf("code %d\n%s\n%s", e.Code, e.Message, e.Details)
}

type MirConnectionError struct {
	Target string
	e      error
}

func (e MirConnectionError) Error() string {
	return fmt.Sprintf("can't establish connection to Mir on %s\n%s", e.Target, e.e)
}

type MirRequestError struct {
	Route string
	e     error
}

func (e MirRequestError) Error() string {
	return fmt.Sprintf("error sending request to Mir on route %s\n%s", e.Route, e.e)
}

type MirResponseError struct {
	Route string
	e     error
}

func (e MirResponseError) Error() string {
	return fmt.Sprintf("error in receiving response from Mir on route %s\n%s", e.Route, e.e)
}

type MirInvalidInputError struct {
	Details []string
}

func (e MirInvalidInputError) Error() string {
	err := "Invalid input\n"
	err += strings.Join(e.Details, "\n")
	return err
}

type MirSerializationError struct {
	Format string
	e      error
}

func (e MirSerializationError) Error() string {
	return fmt.Sprintf("error serializing data to %s\n%s", e.Format, e.e)
}

type MirDeserializationError struct {
	Format string
	e      error
}

func (e MirDeserializationError) Error() string {
	return fmt.Sprintf("error deserializing data to %s\n%s", e.Format, e.e)
}

type MirProcessError struct {
	Msg string
	e   error
}

func (e MirProcessError) Error() string {
	return fmt.Sprintf("error processing request. %s\n%s", e.Msg, e.e)
}

type MirDeviceNotFoundError struct {
	Targets *core_apiv1.Targets
}

func (e MirDeviceNotFoundError) Error() string {
	return fmt.Sprintf("no devices where found with targets. \n%s", e.Targets)
}

type MirEditError struct {
	Msg string
	e   error
}

func (e MirEditError) Error() string {
	return fmt.Sprintf("error editing resource. %s\n%s", e.Msg, e.e)
}

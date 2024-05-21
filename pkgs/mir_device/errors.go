package mir_device

import (
	"fmt"
	"strings"
)

type MirBuilderFieldsError struct {
	Fields []string
}

func (e MirBuilderFieldsError) Error() string {
	return fmt.Sprintf("Fields required for the device is missing or invalid: %s", strings.Join(e.Fields, ", "))
}

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

type MirProcessError struct {
	Msg string
	e   error
}

func (e MirProcessError) Error() string {
	return fmt.Sprintf("error processing request. %s\n%s", e.Msg, e.e)
}

package mir_v1

import "errors"

var (
	// Models Validation
	ErrorInvalidDeviceID             = errors.New("Invalid device ID")
	ErrorDeviceIdAlreadyExist        = errors.New("Device with the same ID or name/namespace already exist")
	ErrorNoDeviceTargetProvided      = errors.New("No device target provided")
	ErrorCommandNameNotProvided      = errors.New("No command name provided")
	ErrorCommandPayloadNotProvided   = errors.New("No command payload provided")
	ErrorCommandEncodingNotSpecified = errors.New("Command encoding not specified")

	// API Requests
	ErrorBadRequest              = errors.New("error occure because of bad request")
	ErrorApiDeserializingRequest = errors.New("error occure while unmarhsalling request")

	// DB Requests
	ErrorDbExecutingQuery        = errors.New("Error occure while executing db query")
	ErrorDbDeserializingResponse = errors.New("Error deserializing db response")
)

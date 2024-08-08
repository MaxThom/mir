package mir_models

import "errors"

var (
	// Models Validation
	ErrorInvalidDeviceID        = errors.New("Invalid device ID")
	ErrorDeviceIdAlreadyExist   = errors.New("Device with the same ID already exist")
	ErrorNoDeviceTargetProvided = errors.New("No device target provided")

	// API Requests
	ErrorApiDeserializingRequest = errors.New("error occure while unmarhsalling request")

	// DB Requests
	ErrorDbExecutingQuery        = errors.New("Error occure while executing db query")
	ErrorDbDeserializingResponse = errors.New("Error deserializing db response")
)

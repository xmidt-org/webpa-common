package context

import (
	"net/http"
)

const (
	// The Content-Type value for JSON
	JsonContentType string = "application/json; charset=UTF-8"

	// The Content-Type header
	ContentTypeHeader string = "Content-Type"
	
	// The X-Content-Type-Options header
	ContentTypeOptionsHeader string = "X-Content-Type-Options"

	// DeviceNameHeader is the standard name of the header which carries the WebPA device
	DeviceNameHeader string = "X-Webpa-Device-Name"

	// MissingDeviceNameHeaderMessage is the error message indicating that the DeviceNameHeader
	// was missing from the request.
	MissingDeviceNameHeaderMessage string = "Missing " + DeviceNameHeader + " header"

	// InvalidDeviceNameHeaderPattern is the format pattern used to create an error message indicating
	// that a device name was improperly formatted.
	InvalidDeviceNameHeaderPattern string = "Invalid " + DeviceNameHeader + " header: %s"
	
	// NoSniff is the value used for content options for errors written by this package
	NoSniff string = "nosniff"
)

var (
	// missingDeviceNameError is an internal HttpError carrying the MissingDeviceNameHeaderMessage
	missingDeviceNameError = NewHttpError(http.StatusBadRequest, MissingDeviceNameHeaderMessage)
)

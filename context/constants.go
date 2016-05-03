package context

const (
	// The Content-Type value for JSON
	JsonContentType string = "application/json; charset=UTF-8"

	// The Content-Type header
	ContentTypeHeader string = "Content-Type"

	// DeviceNameHeader is the standard name of the header which carries the WebPA device
	DeviceNameHeader string = "X-Webpa-Device-Name"

	// MissingDeviceNameHeaderMessage is the error message indicating that the DeviceNameHeader
	// was missing from the request.
	MissingDeviceNameHeaderMessage string = "Missing " + DeviceNameHeader + " header"

	// InvalidDeviceNameHeaderPattern is the format pattern used to create an error message indicating
	// that a device name was improperly formatted.
	InvalidDeviceNameHeaderPattern string = "Invalid " + DeviceNameHeader + " header: %s"

	// ServiceUnavailableMessage is the default message sent when a RequestGate denies a request
	ServiceUnavailableMessage string = "This service is unavailable"
)

var (
	// missingDeviceNameError is an internal HttpError carrying the MissingDeviceNameHeaderMessage
	missingDeviceNameError = NewHttpError(http.StatusBadRequest, MissingDeviceNameHeaderMessage)
)

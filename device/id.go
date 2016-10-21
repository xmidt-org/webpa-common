package device

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/httperror"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// ID represents a normalized identifer for a device.
type ID string

func (id ID) Bytes() []byte {
	return []byte(id)
}

const (
	// InvalidID is a known, global device identifier that is not valid.  Useful
	// when returning errors.
	InvalidID ID = ID("")

	// DefaultDeviceNameHeader is the default header for retrieving the name, which
	// is then parsed into a device ID.
	DefaultDeviceNameHeader = "X-Webpa-Device-Name"

	hexDigits     = "0123456789abcdefABCDEF"
	macDelimiters = ":-.,"
	macPrefix     = "mac"
	macLength     = 12
)

var (
	// idPattern is the precompiled regular expression that all device identifiers must match.
	// Matching is partial, as everything after the service is ignored.
	idPattern = regexp.MustCompile(
		`^(?P<prefix>(?i)mac|uuid|dns|serial):(?P<id>[^/]+)(?P<service>/[^/]+)?`,
	)

	defaultIDHandler = idHandler{
		headerName: DefaultDeviceNameHeader,
		missingHeaderError: httperror.New(
			fmt.Sprintf("Missing header: %s", DefaultDeviceNameHeader),
			http.StatusBadRequest,
			nil,
		),
	}
)

// IDHandler provides the HTTP-related logic for device identifiers.  IDHandler instances
// are safe for concurrent access.
type IDHandler interface {
	// FromValue parses the given value to produce a canonicalized device identifier.
	FromValue(string) (ID, error)

	// FromRequest examines the given HTTP request to produce a canonicalized device
	// identifier.
	FromRequest(*http.Request) (ID, error)

	// ToRequest inserts metadata for the given canonicalized ID into the supplied request.
	// This method is useful for creating client-side requests for connecting to a device Manager.
	ToRequest(ID, *http.Request)
}

// DefaultIDHandler returns the canonicalized IDHandler configured with defaults
func DefaultIDHandler() IDHandler {
	return &defaultIDHandler
}

// NewIDHandler returns an IDHandler using the given header.  If headerName is empty,
// this function simply returns the internal default IDHandler.
func NewIDHandler(headerName string) IDHandler {
	if len(headerName) == 0 {
		return &defaultIDHandler
	}

	return &idHandler{
		headerName: headerName,
		missingHeaderError: httperror.New(
			fmt.Sprintf("Missing header: %s", headerName),
			http.StatusBadRequest,
			nil,
		),
	}
}

// idHandler is the internal IDHandler implementation
type idHandler struct {
	headerName         string
	missingHeaderError error
}

func (h *idHandler) FromValue(value string) (ID, error) {
	match := idPattern.FindStringSubmatch(value)
	if match == nil {
		return InvalidID, errors.New(fmt.Sprintf("Invalid device id: %s", value))
	}

	var (
		prefix  = strings.ToLower(match[1])
		idPart  = match[2]
		service = match[3]
	)

	if prefix == macPrefix {
		var invalidCharacter rune = -1
		idPart = strings.Map(
			func(r rune) rune {
				switch {
				case strings.ContainsRune(hexDigits, r):
					return unicode.ToLower(r)
				case strings.ContainsRune(macDelimiters, r):
					return -1
				default:
					invalidCharacter = r
					return -1
				}
			},
			idPart,
		)

		if invalidCharacter != -1 {
			return InvalidID, errors.New(fmt.Sprintf("Invalid character in mac: %c", invalidCharacter))
		} else if len(idPart) != macLength {
			return InvalidID, errors.New(fmt.Sprintf("Invalid length of mac: %s", idPart))
		}
	}

	if len(service) > 0 {
		return ID(fmt.Sprintf("%s:%s%s/", prefix, idPart, service)), nil
	}

	return ID(fmt.Sprintf("%s:%s", prefix, idPart)), nil
}

func (h *idHandler) FromRequest(request *http.Request) (ID, error) {
	deviceName := request.Header.Get(h.headerName)
	if len(deviceName) == 0 {
		return InvalidID, h.missingHeaderError
	}

	return h.FromValue(deviceName)
}

func (h *idHandler) ToRequest(id ID, request *http.Request) {
	request.Header.Set(h.headerName, string(id))
}

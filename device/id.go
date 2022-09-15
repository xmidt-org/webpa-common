package device

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// ID represents a normalized identifier for a device.
type ID string

// Bytes is a convenience function to obtain the []byte representation of an ID.
func (id ID) Bytes() []byte {
	return []byte(id)
}

const (
	hexDigits     = "0123456789abcdefABCDEF"
	macDelimiters = ":-.,"
	macPrefix     = "mac"
	macLength     = 12
)

var (
	invalidID = ID("")

	// idPattern is the precompiled regular expression that all device identifiers must match.
	// Matching is partial, as everything after the service is ignored.
	idPattern = regexp.MustCompile(
		`^(?P<prefix>(?i)mac|uuid|dns|serial):(?P<id>[^/]+)(?P<service>/[^/]+)?`,
	)
)

// IntToMAC accepts a 64-bit integer and formats that as a device MAC address identifier
// The returned ID will be of the form mac:XXXXXXXXXXXX, where X is a hexadecimal digit using
// lowercased letters.
func IntToMAC(value uint64) ID {
	return ID(fmt.Sprintf("mac:%012x", value&0x0000FFFFFFFFFFFF))
}

// ParseID parses a raw device name into a canonicalized identifier.
func ParseID(deviceName string) (ID, error) {
	match := idPattern.FindStringSubmatch(deviceName)
	if match == nil {
		return invalidID, ErrorInvalidDeviceName
	}

	var (
		prefix = strings.ToLower(match[1])
		idPart = match[2]
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

		if invalidCharacter != -1 || len(idPart) != macLength {
			return invalidID, ErrorInvalidDeviceName
		}
	}

	return ID(fmt.Sprintf("%s:%s", prefix, idPart)), nil
}

// IDHashParser is a parsing function that examines an HTTP request to produce
// a []byte key for consistent hashing.  The returned function examines the
// given request header and invokes ParseID on the value.
//
// If deviceNameHeader is the empty string, DefaultDeviceNameHeader is used.
func IDHashParser(request *http.Request) ([]byte, error) {
	deviceName := request.Header.Get(DeviceNameHeader)
	if len(deviceName) == 0 {
		return nil, ErrorMissingDeviceNameHeader
	}

	id, err := ParseID(deviceName)
	if err != nil {
		return nil, err
	}

	return id.Bytes(), nil
}

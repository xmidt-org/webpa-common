package device

import (
	"errors"
	"fmt"
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

func ParseID(value string) (ID, error) {
	match := idPattern.FindStringSubmatch(value)
	if match == nil {
		return invalidID, errors.New(fmt.Sprintf("Invalid device id: %s", value))
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
			return invalidID, errors.New(fmt.Sprintf("Invalid character in mac: %c", invalidCharacter))
		} else if len(idPart) != macLength {
			return invalidID, errors.New(fmt.Sprintf("Invalid length of mac: %s", idPart))
		}
	}

	if len(service) > 0 {
		return ID(fmt.Sprintf("%s:%s%s/", prefix, idPart, service)), nil
	}

	return ID(fmt.Sprintf("%s:%s", prefix, idPart)), nil
}

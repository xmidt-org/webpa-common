// Package canonical provides a simple normalization for device identifiers.  These
// identifiers have the form {prefix}:{id}/{service}/{ignored}, where only {prefix}
// and {id} are required.
package canonical

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Id represents a normalized identifer for a device.
type Id interface {
	// Bytes returns a distinct byte slice representation
	// of this canonicalized identifier.
	Bytes() []byte
}

// id is the internal Id implementation
type id string

func (this id) Bytes() []byte {
	return []byte(this)
}

var _ Id = id("")

const (
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
)

// ParseId parses a raw string identifier into an Id
func ParseId(value string) (Id, error) {
	match := idPattern.FindStringSubmatch(value)
	if match == nil {
		return nil, errors.New(fmt.Sprintf("Invalid device id: %s", value))
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
			return nil, errors.New(fmt.Sprintf("Invalid character in mac: %c", invalidCharacter))
		} else if len(idPart) != macLength {
			return nil, errors.New(fmt.Sprintf("Invalid length of mac: %s", idPart))
		}
	}

	if len(service) > 0 {
		return id(fmt.Sprintf("%s:%s%s/", prefix, idPart, service)), nil
	}

	return id(fmt.Sprintf("%s:%s", prefix, idPart)), nil
}

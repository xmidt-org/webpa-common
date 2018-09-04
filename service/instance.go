package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DefaultScheme is the default URI scheme assumed by this service discovery infrastructure
const DefaultScheme = "https"

// FormatInstance creates an instance string from a (scheme, address, port) tuple.  If the port is the default
// for the scheme, it is not included.  If scheme is empty, DefaultScheme is used.  If the port is nonpositive,
// then it is ignored.
func FormatInstance(scheme, address string, port int) string {
	if len(scheme) == 0 {
		scheme = DefaultScheme
	}

	if port > 0 {
		includePort := true
		switch strings.ToLower(scheme) {
		case "http":
			includePort = (port != 80)
		case "https":
			includePort = (port != 443)
		}

		if includePort {
			return fmt.Sprintf("%s://%s:%d", scheme, address, port)
		}
	}

	return fmt.Sprintf("%s://%s", scheme, address)
}

// instancePattern is what NormalizeInstance expects to be matched.  This pattern is intentionally liberal, and allows
// URIs that are disallowed under https://www.ietf.org/rfc/rfc2396.txt
var instancePattern = regexp.MustCompile("^((?P<scheme>.+)://)?(?P<address>[^:]+)(:(?P<port>[0-9]+))?$")

// NormalizeInstance canonicalizes an instance string from a service discovery backend, using an optional defaultScheme to
// be used if no scheme is found in the instance string.  If defaultScheme is empty, DefaultScheme is assumed instead.
//
// This function performs the following on the instance:
//   (1) If instance is a blank string, e.g. contains only whitespace or is empty, an empty string is returned with an error
//   (2) If the instance with whitespace trimmed off is not a valid instance, an error is returned with the trimmed instance string.
//       This function is intentionally lenient on what is a valid instance string, e.g. "foobar.com", "foobar.com:8080", "asdf://foobar.com", etc
//   (3) If there was no scheme prepended on the instance, either defaultScheme (if non-empty) or the global DefaultScheme is used instead
//   (4) Finally, FormatInstance is called with the parsed scheme, address, and port.  Default ports for schemes will be omitted from the
//       final string.
func NormalizeInstance(defaultScheme, instance string) (string, error) {
	instance = strings.TrimSpace(instance)
	if len(instance) == 0 {
		return instance, errors.New("Blank instances are not allowed")
	}

	submatches := instancePattern.FindStringSubmatch(instance)
	if len(submatches) == 0 {
		return instance, fmt.Errorf("Invalid instance: %s", instance)
	}

	var (
		scheme  = submatches[2]
		address = submatches[3]
	)

	if len(scheme) == 0 {
		if len(defaultScheme) > 0 {
			scheme = defaultScheme
		} else {
			scheme = DefaultScheme
		}
	}

	var port int
	if portValue := submatches[5]; len(portValue) > 0 {
		var err error
		port, err = strconv.Atoi(submatches[5])
		if err != nil {
			// NOTE: Shouldn't ever hit this case, because the port is constrained by the regexp to be numeric
			return instance, err
		}
	}

	return FormatInstance(scheme, address, port), nil
}

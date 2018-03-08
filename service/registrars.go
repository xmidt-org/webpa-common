package service

import (
	"fmt"
	"strings"

	"github.com/go-kit/kit/sd"
)

// FormatURL creates a URL from a (scheme, address, port) tuple.  If the port is the default
// for the scheme, it is not included.
func FormatURL(scheme, address string, port int) string {
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

	return fmt.Sprintf("%s://%s", scheme, address)
}

// NopRegistrar is an sd.Registrar that simply does nothing.  Useful as an alternative to nil.
type NopRegistrar struct{}

func (nr NopRegistrar) Register()   {}
func (nr NopRegistrar) Deregister() {}

// Registrars is a aggregate sd.Registrar that allows allows composite registration and deregistration.
type Registrars []sd.Registrar

func (r Registrars) Register() {
	for _, v := range r {
		v.Register()
	}
}

func (r Registrars) Deregister() {
	for _, v := range r {
		v.Deregister()
	}
}

func (r Registrars) Len() int {
	return len(r)
}

func (r *Registrars) Add(v ...sd.Registrar) {
	*r = append(*r, v...)
}

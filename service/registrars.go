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

// Registrars is a aggregate sd.Registrar that allows for named registrars keyed
// by some string, usually the service URL.  The zero value can be used as is.
type Registrars map[string]sd.Registrar

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

func (r Registrars) Has(key string) bool {
	_, ok := r[key]
	return ok
}

func (r Registrars) Get(key string) (sd.Registrar, bool) {
	v, ok := r[key]
	return v, ok
}

func (r *Registrars) Set(key string, value sd.Registrar) {
	if *r == nil {
		*r = make(Registrars)
	}

	(*r)[key] = value
}

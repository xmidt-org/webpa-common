package service

import (
	"github.com/go-kit/kit/sd"
)

// Registrars is a aggregate sd.Registrar that allows allows composite registration and deregistration.
// Keys in this map type will be service advertisements or instances, e.g. "host.com:8080" or "https://foobar.com"
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

func (r Registrars) Has(key string) bool {
	_, ok := r[key]
	return ok
}

func (r Registrars) Len() int {
	return len(r)
}

func (r *Registrars) Add(key string, v sd.Registrar) {
	if *r == nil {
		*r = make(Registrars)
	}

	(*r)[key] = v
}

package service

import (
	"github.com/go-kit/kit/sd"
)

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

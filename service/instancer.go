package service

import "github.com/go-kit/kit/sd"

// Instancers is a collection of sd.Instancer objects, keyed by arbitrary strings.
// Typically, the keys will be paths within the service discovery backend.
type Instancers map[string]sd.Instancer

func (i Instancers) Len() int {
	return len(i)
}

func (i Instancers) Has(key string) bool {
	_, ok := i[key]
	return ok
}

func (i Instancers) Get(key string) (sd.Instancer, bool) {
	v, ok := i[key]
	return v, ok
}

func (i *Instancers) Set(key string, value sd.Instancer) {
	if *i == nil {
		*i = make(Instancers)
	}

	(*i)[key] = value
}

func (i Instancers) Stop() {
	for _, v := range i {
		v.Stop()
	}
}

package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

type instancerEntry struct {
	l log.Logger
	i sd.Instancer
}

// Instancers is a collection of sd.Instancer objects, keyed by arbitrary strings.
type Instancers map[string]instancerEntry

func (is Instancers) Len() int {
	return len(is)
}

func (is Instancers) Has(key string) bool {
	_, ok := is[key]
	return ok
}

func (is Instancers) Get(key string) (log.Logger, sd.Instancer, bool) {
	v, ok := is[key]
	return v.l, v.i, ok
}

func (is *Instancers) Set(key string, l log.Logger, i sd.Instancer) {
	if *is == nil {
		*is = make(Instancers)
	}

	if l == nil {
		l = logging.DefaultLogger()
	}

	(*is)[key] = instancerEntry{l, i}
}

func (is Instancers) Each(f func(string, log.Logger, sd.Instancer)) {
	for k, v := range is {
		f(k, v.l, v.i)
	}
}

func (is Instancers) Copy() (clone Instancers) {
	if len(is) > 0 {
		clone = make(Instancers, len(is))
		for k, v := range is {
			clone[k] = v
		}
	}

	return
}

func (is Instancers) Stop() {
	for _, v := range is {
		v.i.Stop()
	}
}

// NewFixedInstancers is a convenience for creating an Instancers with a single sd.Instancer whose
// instances are constant, i.e. not updated by any service discovery backend.  If the logger is nil,
// a default logger is used.  The logger associated with the instancer will be augmented with
// contextual information.
func NewFixedInstancers(l log.Logger, key string, i []string) Instancers {
	if l == nil {
		l = logging.DefaultLogger()
	}

	var is Instancers
	is.Set(
		key,
		log.With(l, "fixedInstances", i),
		sd.FixedInstancer(i),
	)

	return is
}

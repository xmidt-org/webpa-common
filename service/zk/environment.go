package zk

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	gokitzk "github.com/go-kit/kit/sd/zk"
)

func newService(r Registration) (string, gokitzk.Service) {
	key := service.FormatURL(
		r.scheme(),
		r.address(),
		r.port(),
	)

	return key, gokitzk.Service{
		Path: r.path(),
		Name: r.name(),
		Data: []byte(key),
	}
}

func newClient(l log.Logger, zo Options) (gokitzk.Client, error) {
	return gokitzk.NewClient(
		zo.servers(),
		l,
		gokitzk.ConnectTimeout(zo.connectTimeout()),
		gokitzk.SessionTimeout(zo.sessionTimeout()),
	)
}

func newInstancers(l log.Logger, c gokitzk.Client, zo Options) (i service.Instancers, err error) {
	for _, path := range zo.watches() {
		if i.Has(path) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "path", path)
			continue
		}

		var instancer sd.Instancer
		instancer, err = gokitzk.NewInstancer(c, path, l)
		if err != nil {
			return
		}

		i.Set(path, instancer)
	}

	return
}

func newRegistrars(l log.Logger, c gokitzk.Client, zo Options) (r service.Registrars) {
	for _, registration := range zo.registrations() {
		k, s := newService(registration)
		if r.Has(k) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate registration", "url", k)
			continue
		}

		r.Set(k, gokitzk.NewRegistrar(c, s, l))
	}

	return
}

// NewEnvironment constructs a Zookeeper-based service.Environment using both a zookeeper Options (typically unmarshaled
// from configuration) and an optional extra set of environment options.
func NewEnvironment(l log.Logger, zo Options, eo ...service.EnvironmentOption) (service.Environment, error) {
	if l == nil {
		l = logging.DefaultLogger()
	}

	if len(zo.Watches) == 0 && len(zo.Registrations) == 0 {
		return nil, nil
	}

	c, err := newClient(l, zo)
	if err != nil {
		return nil, err
	}

	i, err := newInstancers(l, c, zo)
	if err != nil {
		c.Stop()
		return nil, err
	}

	r := newRegistrars(l, c, zo)

	eo = append(
		eo,
		service.WithRegistrars(r),
		service.WithInstancers(i),
		service.WithCloser(func() error { c.Stop(); return nil }),
	)

	return service.NewEnvironment(eo...), nil
}

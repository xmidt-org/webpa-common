package zk

import (
	"go.uber.org/zap"

	"github.com/go-kit/kit/sd"
	gokitzk "github.com/go-kit/kit/sd/zk"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/adapter"
	"github.com/xmidt-org/webpa-common/v2/service"
)

func newService(r Registration) (string, gokitzk.Service) {
	url := service.FormatInstance(
		r.scheme(),
		r.address(),
		r.port(),
	)
	return url, gokitzk.Service{
		Path: r.path(),
		Name: r.name(),
		Data: []byte(url),
	}
}

// clientFactory is the factory function used to create a go-kit zookeeper Client.
// Tests can change this for mocked behavior.
var clientFactory = gokitzk.NewClient
var logger = adapter.Logger{
	Logger: sallust.Default(),
}

func newClient(l *zap.Logger, zo Options) (gokitzk.Client, error) {
	client := zo.client()
	return clientFactory(
		client.servers(),
		logger,
		gokitzk.ConnectTimeout(client.connectTimeout()),
		gokitzk.SessionTimeout(client.sessionTimeout()),
	)
}

func newInstancer(l *zap.Logger, c gokitzk.Client, path string) (i sd.Instancer, err error) {
	i, err = gokitzk.NewInstancer(c, path, logger)
	if err == nil {
		i = service.NewContextualInstancer(i, map[string]interface{}{"path": path})
	}

	return
}

func newInstancers(l *zap.Logger, c gokitzk.Client, zo Options) (i service.Instancers, err error) {
	for _, path := range zo.watches() {
		if i.Has(path) {
			l.Warn("skipping duplicate watch", zap.String("path", path))
			continue
		}

		var instancer sd.Instancer
		instancer, err = newInstancer(l, c, path)
		if err != nil {
			// ensure the previously create instancers are stopped
			i.Stop()
			return
		}
		i.Set(path, instancer)
	}
	return
}

func newRegistrars(base *zap.Logger, c gokitzk.Client, zo Options) (r service.Registrars) {
	for _, registration := range zo.registrations() {
		instance, s := newService(registration)
		if r.Has(instance) {
			base.Warn("skipping duplicate registration", zap.String("instance", instance))
			continue
		}

		r.Add(instance, gokitzk.NewRegistrar(c, s, logger))
	}

	return
}

// NewEnvironment constructs a Zookeeper-based service.Environment using both a zookeeper Options (typically unmarshaled
// from configuration) and an optional extra set of environment options.
func NewEnvironment(l *zap.Logger, zo Options, eo ...service.Option) (service.Environment, error) {
	if l == nil {
		l = sallust.Default()
	}

	if len(zo.Watches) == 0 && len(zo.Registrations) == 0 {
		return nil, service.ErrIncomplete
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
	return service.NewEnvironment(
		append(
			eo,
			service.WithRegistrars(newRegistrars(l, c, zo)),
			service.WithInstancers(i),
			service.WithCloser(func() error { c.Stop(); return nil }),
		)...,
	), nil
}

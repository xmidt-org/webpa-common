package service

import (
	"sync/atomic"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/zk"
)

// Interface represents a service discovery facade.  It's a very thin layer
// on top of a go-kit/kit/sd subpackage.
type Interface interface {
	sd.Registrar

	// NewInstancer creates an sd.Instancer appropriate for listening for service
	// changes.  Note that this only supports (1) service at this time.
	NewInstancer() (sd.Instancer, error)

	// Close shuts down this facade.  Calling any other method on this instance after
	// a call to this method is undefined.  However, this method is itself idempotent.
	Close() error
}

// zkFacade is the facade for go-kit/kit/sd/zk
type zkFacade struct {
	logger    log.Logger
	state     uint32
	client    zk.Client
	path      string
	registrar sd.Registrar
}

func (z *zkFacade) Register() {
	if z.registrar != nil {
		z.registrar.Register()
	}
}

func (z *zkFacade) Deregister() {
	if z.registrar != nil {
		z.registrar.Deregister()
	}
}

func (z *zkFacade) NewInstancer() (sd.Instancer, error) {
	return zk.NewInstancer(
		z.client,
		z.path,
		z.logger,
	)
}

func (z *zkFacade) Close() error {
	if atomic.CompareAndSwapUint32(&z.state, 0, 1) {
		z.Deregister()
		z.client.Stop()
	}

	return nil
}

// New constructs a service discovery facade from a set of Options.
func New(o *Options) (Interface, error) {
	var (
		path      = o.path()
		registrar sd.Registrar
		logger    = logging.DefaultCaller(o.logger(), "service", true, "path", path)

		client, err = zk.NewClient(
			o.servers(),
			logger,
			zk.ConnectTimeout(o.connectTimeout()),
			zk.SessionTimeout(o.sessionTimeout()),
		)
	)

	if err != nil {
		return nil, err
	}

	registration := o.registration()
	if len(registration) > 0 {
		registrar = zk.NewRegistrar(
			client,
			zk.Service{
				Path: path,
				Name: o.serviceName(),
				Data: []byte(registration),
			},
			logger,
		)
	}

	return &zkFacade{
		logger:    logger,
		client:    client,
		path:      path,
		registrar: registrar,
	}, nil
}

package consul

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/sd"
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/util/conn"
	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/webpa-common/v2/adapter"
	"github.com/xmidt-org/webpa-common/v2/service"
	"go.uber.org/zap"
)

var (
	errNoDatacenters = errors.New("could not acquire datacenters")
)

// Environment is a consul-specific interface for the service discovery environment.
// A primary use case is obtaining access to the underlying consul client for use
// in direct API calls.
type Environment interface {
	service.Environment

	// Client returns the custom consul Client interface exposed by this package
	Client() Client
}

type environment struct {
	service.Environment
	client Client
}

func (e environment) Client() Client {
	return e.client
}

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// TODO: When does this ever happen?
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func ensureIDs(r *api.AgentServiceRegistration) {
	if len(r.ID) == 0 {
		r.ID = generateID()
	}
	if r.Check != nil && len(r.Check.CheckID) == 0 {
		r.Check.CheckID = generateID()
	}
	for _, check := range r.Checks {
		if len(check.CheckID) == 0 {
			check.CheckID = generateID()
		}
	}
}

func newInstancerKey(w Watch) string {
	return fmt.Sprintf(
		"%s%s{passingOnly=%t}{datacenter=%s}",
		w.Service,
		w.Tags,
		w.PassingOnly,
		w.QueryOptions.Datacenter,
	)
}

func defaultClientFactory(client *api.Client) (Client, ttlUpdater) {
	return NewClient(client), client.Agent()
}

var clientFactory = defaultClientFactory

func getDatacenters(l *zap.Logger, c Client, co Options) ([]string, error) {
	datacenters, err := c.Datacenters()
	if err == nil {
		return datacenters, nil
	}

	l.Error("Could not acquire datacenters on initial attempt", zap.Error(err))

	d := 30 * time.Millisecond
	for retry := 0; retry < co.datacenterRetries(); retry++ {
		time.Sleep(d)
		d = conn.Exponential(d)
		datacenters, err = c.Datacenters()
		if err == nil {
			return datacenters, nil
		}

		l.Error("Could not acquire datacenters", zap.Int("retryCount", retry), zap.Error(err))
	}

	return nil, errNoDatacenters
}

func newInstancer(l *zap.Logger, c Client, w Watch) sd.Instancer {
	return service.NewContextualInstancer(
		NewInstancer(InstancerOptions{
			Client:       c,
			Logger:       l,
			Service:      w.Service,
			Tags:         w.Tags,
			PassingOnly:  w.PassingOnly,
			QueryOptions: w.QueryOptions,
		}),
		map[string]interface{}{
			"service":     w.Service,
			"tags":        w.Tags,
			"passingOnly": w.PassingOnly,
			"datacenter":  w.QueryOptions.Datacenter,
		},
	)
}

func newInstancers(l *zap.Logger, c Client, co Options) (i service.Instancers, err error) {
	var datacenters []string
	for _, w := range co.watches() {
		if w.CrossDatacenter {
			if len(datacenters) == 0 {
				datacenters, err = getDatacenters(l, c, co)
				if err != nil {
					return
				}
			}
			for _, datacenter := range datacenters {
				w.QueryOptions.Datacenter = datacenter
				key := newInstancerKey(w)
				if i.Has(key) {
					l.Warn("skipping duplicate watch", zap.String("service", w.Service), zap.Strings("tags", w.Tags), zap.Bool("passingOnly", w.PassingOnly), zap.String("datacenter", w.QueryOptions.Datacenter))
					continue
				}
				i.Set(key, newInstancer(l, c, w))
			}
		} else {
			key := newInstancerKey(w)
			if i.Has(key) {
				l.Warn("skipping duplicate watch", zap.String("service", w.Service), zap.Strings("tags", w.Tags), zap.Bool("passingOnly", w.PassingOnly), zap.String("datacenter", w.QueryOptions.Datacenter))
				continue
			}
			i.Set(key, newInstancer(l, c, w))
		}
	}
	return
}

func newRegistrars(l *adapter.Logger, registrationScheme string, c gokitconsul.Client, u ttlUpdater, co Options) (r service.Registrars, closer func() error, err error) {
	var consulRegistrar sd.Registrar
	for _, registration := range co.registrations() {
		instance := service.FormatInstance(registrationScheme, registration.Address, registration.Port)
		if r.Has(instance) {
			l.Logger.Warn("skipping duplicate registration", zap.String("instance", instance))
			continue
		}

		if !co.disableGenerateID() {
			ensureIDs(&registration)
		}
		rid := zap.String("id", registration.ID)
		in := zap.String("instance", instance)
		l.Logger = l.Logger.With(rid, in)
		consulRegistrar, err = NewRegistrar(c, u, registration, l)
		if err != nil {
			return
		}
		r.Add(instance, consulRegistrar)
	}
	return
}

func NewEnvironment(l *adapter.Logger, registrationScheme string, co Options, eo ...service.Option) (service.Environment, error) {
	if l == nil {
		l = adapter.DefaultLogger()
	}

	if len(co.Watches) == 0 && len(co.Registrations) == 0 {
		return nil, service.ErrIncomplete
	}
	consulClient, err := api.NewClient(co.config())
	if err != nil {
		return nil, err
	}
	client, updater := clientFactory(consulClient)
	r, closer, err := newRegistrars(l, registrationScheme, client, updater, co)
	if err != nil {
		return nil, err
	}
	i, err := newInstancers(l.Logger, client, co)
	if err != nil {
		return nil, err
	}
	newServiceEnvironment := environment{
		service.NewEnvironment(
			append(
				eo,
				service.WithRegistrars(r),
				service.WithInstancers(i),
				service.WithCloser(closer),
			)...), NewClient(consulClient)}
	if co.DatacenterWatchInterval > 0 || (len(co.Chrysom.Bucket) > 0 && co.Chrysom.Listen.PullInterval > 0) {
		_, err := newDatacenterWatcher(l.Logger, newServiceEnvironment, co)
		if err != nil {
			l.Logger.Error("Could not create datacenter watcher", zap.Error(err))
		}
	}

	return newServiceEnvironment, nil
}

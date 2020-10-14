package consul

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/util/conn"
	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

var (
	errNoDatacenters = errors.New("Could not acquire datacenters")
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

func getDatacenters(l log.Logger, c Client, co Options) ([]string, error) {
	datacenters, err := c.Datacenters()
	if err == nil {
		return datacenters, nil
	}

	l.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not acquire datacenters on initial attempt", logging.ErrorKey(), err)

	d := 30 * time.Millisecond
	for retry := 0; retry < co.datacenterRetries(); retry++ {
		time.Sleep(d)
		d = conn.Exponential(d)

		datacenters, err = c.Datacenters()
		if err == nil {
			return datacenters, nil
		}

		l.Log(level.Key(), level.ErrorValue(), "retryCount", retry, logging.MessageKey(), "Could not acquire datacenters", logging.ErrorKey(), err)
	}

	return nil, errNoDatacenters
}

func newInstancer(l log.Logger, c Client, w Watch) sd.Instancer {
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

func newInstancers(l log.Logger, c Client, co Options) (i service.Instancers, err error) {
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
					l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly, "datacenter", w.QueryOptions.Datacenter)
					continue
				}
				i.Set(key, newInstancer(l, c, w))
			}
		} else {
			key := newInstancerKey(w)
			if i.Has(key) {
				l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly, "datacenter", w.QueryOptions.Datacenter)
				continue
			}
			i.Set(key, newInstancer(l, c, w))
		}
	}

	return
}

func newRegistrars(l log.Logger, registrationScheme string, c gokitconsul.Client, u ttlUpdater, co Options) (r service.Registrars, closer func() error, err error) {
	var consulRegistrar sd.Registrar
	for _, registration := range co.registrations() {
		instance := service.FormatInstance(registrationScheme, registration.Address, registration.Port)
		if r.Has(instance) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate registration", "instance", instance)
			continue
		}

		if !co.disableGenerateID() {
			ensureIDs(&registration)
		}

		consulRegistrar, err = NewRegistrar(c, u, &registration, log.With(l, "id", registration.ID, "instance", instance))
		if err != nil {
			return
		}

		r.Add(instance, consulRegistrar)
	}

	return
}

func NewEnvironment(l log.Logger, registrationScheme string, co Options, eo ...service.Option) (service.Environment, error) {
	if l == nil {
		l = logging.DefaultLogger()
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

	i, err := newInstancers(l, client, co)
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

	if co.DatacenterWatchInterval > 0 {
		go WatchInstancers(l, co, newServiceEnvironment)
	}

	return newServiceEnvironment, nil
}

func WatchInstancers(l log.Logger, co Options, e Environment) {
	if co.DatacenterWatchInterval <= 0 {
		l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "Not setting up instancer watch. Watch interval: ", co.DatacenterWatchInterval)
		return
	}

	createInstancersAgain := time.NewTicker(co.DatacenterWatchInterval)
	for {
		select {
		case <-createInstancersAgain.C:
			//TODO: check for error when creating new instancers and log that error if it happens. Don't reset instancers
			//if newInstancers returns nil
			instancers, err := newInstancers(l, e.Client(), co)

			if err != nil {
				l.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not refresh instancers",
					logging.ErrorKey(), err)
				return
			}
			e.SetInstancers(instancers)
		}
	}
}

package consul

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

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
		"%s%s{passingOnly=%t}",
		w.Service,
		w.Tags,
		w.PassingOnly,
	)
}

func defaultClientFactory(client *api.Client) (gokitconsul.Client, ttlUpdater) {
	return gokitconsul.NewClient(client), client.Agent()
}

var clientFactory = defaultClientFactory

func newClient(co Options) (gokitconsul.Client, ttlUpdater, error) {
	consulClient, err := api.NewClient(co.config())
	if err != nil {
		return nil, nil, err
	}

	gokitClient, updater := clientFactory(consulClient)
	return gokitClient, updater, nil
}

func newInstancer(l log.Logger, c gokitconsul.Client, w Watch) sd.Instancer {
	return service.NewContextualInstancer(
		gokitconsul.NewInstancer(
			c,
			log.With(l, "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly),
			w.Service,
			w.Tags,
			w.PassingOnly,
		),
		map[string]interface{}{
			"service":     w.Service,
			"tags":        w.Tags,
			"passingOnly": w.PassingOnly,
		},
	)
}

func newInstancers(l log.Logger, c gokitconsul.Client, co Options) (i service.Instancers) {
	for _, w := range co.watches() {
		key := newInstancerKey(w)
		if i.Has(key) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly)
			continue
		}

		i.Set(key, newInstancer(l, c, w))
	}

	return
}

func newRegistrars(l log.Logger, registrationScheme string, c gokitconsul.Client, u ttlUpdater, co Options) (r service.Registrars, closer func() error, err error) {
	var consulRegistrar sd.Registrar
	for _, registration := range co.registrations() {
		localRegistration := registration

		instance := service.FormatInstance(registrationScheme, localRegistration.Address, localRegistration.Port)
		if r.Has(instance) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate registration", "instance", instance)
			continue
		}

		if !co.disableGenerateID() {
			ensureIDs(&localRegistration)
		}

		consulRegistrar, err = NewRegistrar(c, u, &localRegistration, log.With(l, "id", localRegistration.ID, "instance", instance))
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
		return nil, nil
	}

	c, u, err := newClient(co)
	if err != nil {
		return nil, err
	}

	r, closer, err := newRegistrars(l, registrationScheme, c, u, co)
	if err != nil {
		return nil, err
	}

	return service.NewEnvironment(
		append(
			eo,
			service.WithRegistrars(r),
			service.WithInstancers(newInstancers(l, c, co)),
			service.WithCloser(closer),
		)...,
	), nil
}

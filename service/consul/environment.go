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

func generateServiceID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// TODO: When does this ever happen?
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(b)
}

func newInstancerKey(w Watch) string {
	return fmt.Sprintf(
		"%s%s{passingOnly=%t}",
		w.Service,
		w.Tags,
		w.PassingOnly,
	)
}

var clientFactory = gokitconsul.NewClient

func newClient(co Options) (gokitconsul.Client, error) {
	cc, err := api.NewClient(co.config())
	if err != nil {
		return nil, err
	}

	return clientFactory(cc), nil
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

func newRegistrars(l log.Logger, c gokitconsul.Client, co Options) (r service.Registrars) {
	for _, registration := range co.registrations() {
		instance := fmt.Sprintf("%s:%d", registration.Address, registration.Port)
		if r.Has(instance) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate registration", "instance", instance)
			continue
		}

		if len(registration.ID) == 0 && !co.disableGenerateID() {
			registration.ID = generateServiceID()
		}

		r.Add(
			instance,
			gokitconsul.NewRegistrar(c, &registration, log.With(l, "id", registration.ID, "instance", instance)),
		)
	}

	return
}

func NewEnvironment(l log.Logger, co Options, eo ...service.Option) (service.Environment, error) {
	if l == nil {
		l = logging.DefaultLogger()
	}

	if len(co.Watches) == 0 && len(co.Registrations) == 0 {
		return nil, nil
	}

	c, err := newClient(co)
	if err != nil {
		return nil, err
	}

	return service.NewEnvironment(
		append(
			eo,
			service.WithRegistrars(newRegistrars(l, c, co)),
			service.WithInstancers(newInstancers(l, c, co)),
		)...,
	), nil
}

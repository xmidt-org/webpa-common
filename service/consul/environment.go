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

func newClient(co Options) (gokitconsul.Client, error) {
	cc, err := api.NewClient(co.config())
	if err != nil {
		return nil, err
	}

	return gokitconsul.NewClient(cc), nil
}

func newInstancer(base log.Logger, c gokitconsul.Client, w Watch) (l log.Logger, i sd.Instancer) {
	l = log.With(l, "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly)
	i = gokitconsul.NewInstancer(c, l, w.Service, w.Tags, w.PassingOnly)
	return
}

func newInstancers(base log.Logger, c gokitconsul.Client, co Options) (i service.Instancers) {
	for _, w := range co.watches() {
		key := newInstancerKey(w)
		if i.Has(key) {
			base.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly)
			continue
		}

		logger, instancer := newInstancer(base, c, w)
		i.Set(key, logger, instancer)
	}

	return
}

func newRegistrars(base log.Logger, c gokitconsul.Client, co Options) (r service.Registrars) {
	dedupe := make(map[string]bool)
	for _, registration := range co.registrations() {
		endpoint := fmt.Sprintf("%s:%d", registration.Address, registration.Port)
		if dedupe[endpoint] {
			base.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate registration", "endpoint", endpoint)
			continue
		}

		dedupe[endpoint] = true
		if len(registration.ID) == 0 && !co.disableGenerateID() {
			registration.ID = generateServiceID()
		}

		r.Add(
			gokitconsul.NewRegistrar(c, &registration, log.With(base, "id", registration.ID, "endpoint", endpoint)),
		)
	}

	return
}
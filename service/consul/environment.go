package consul

import (
	"bytes"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

func newKey(w Watch) string {
	o := bytes.NewBufferString(w.Service)

	return o.String()
}

func newClient(co Options) (gokitconsul.Client, error) {
	cc, err := api.NewClient(co.config())
	if err != nil {
		return nil, err
	}

	return gokitconsul.NewClient(cc), nil
}

func newInstancers(l log.Logger, c gokitconsul.Client, co Options) (i service.Instancers) {
	for _, w := range co.watches() {
		key := newKey(w)
		if i.Has(key) {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly)
			continue
		}

		i.Set(key, gokitconsul.NewInstancer(c, l, w.Service, w.Tags, w.PassingOnly))
	}

	return
}

package servicecfg

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/zk"
	"github.com/Comcast/webpa-common/xviper"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
)

func NewEnvironment(l log.Logger, u xviper.Unmarshaler) (service.Environment, error) {
	if l == nil {
		l = logging.DefaultLogger()
	}

	o := new(Options)
	if err := u.Unmarshal(&o); err != nil {
		return nil, err
	}

	if len(o.Fixed) > 0 {
		l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using a fixed set of instances for service discovery", "instances", o.Fixed)
		return service.NewEnvironment(
			service.WithVnodeCount(o.vnodeCount()),
			service.WithInstancers(service.Instancers{"fixed": sd.FixedInstancer(o.Fixed)}),
		), nil
	}

	if o.Zookeeper != nil {
		l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using zookeeper for service discovery")
		return zk.NewEnvironment(l, *o.Zookeeper, service.WithVnodeCount(o.vnodeCount()))
	}

	/*
		if o.Consul != nil {
			l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using consul for service discovery")
			return consul.NewEnvironment(l, *o.Consul)
		}
	*/

	return nil, nil
}

package servicecfg

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/consul"
	"github.com/Comcast/webpa-common/service/zk"
	"github.com/Comcast/webpa-common/xviper"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
)

var (
	zookeeperEnvironmentFactory = zk.NewEnvironment
	consulEnvironmentFactory    = consul.NewEnvironment
)

func NewEnvironment(l log.Logger, u xviper.Unmarshaler, options ...service.Option) (service.Environment, error) {
	if l == nil {
		l = logging.DefaultLogger()
	}

	o := new(Options)
	if err := u.Unmarshal(&o); err != nil {
		return nil, err
	}

	eo := []service.Option{
		service.WithAccessorFactory(
			service.NewConsistentAccessorFactory(o.vnodeCount()),
		),
		service.WithDefaultScheme(o.defaultScheme()),
	}

	eo = append(eo, options...)

	if len(o.Fixed) > 0 {
		l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using a fixed set of instances for service discovery", "instances", o.Fixed)
		return service.NewEnvironment(
			append(eo,
				service.WithInstancers(
					service.Instancers{
						"fixed": service.NewContextualInstancer(
							sd.FixedInstancer(o.Fixed),
							map[string]interface{}{"fixed": o.Fixed},
						),
					},
				),
			)...,
		), nil
	}

	if o.Zookeeper != nil {
		l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using zookeeper for service discovery")
		return zookeeperEnvironmentFactory(l, *o.Zookeeper, eo...)
	}

	if o.Consul != nil {
		l.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "using consul for service discovery")
		return consulEnvironmentFactory(l, o.DefaultScheme, *o.Consul, eo...)
	}

	return nil, nil
}

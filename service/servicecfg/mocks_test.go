package servicecfg

import (
	"github.com/Comcast/webpa-common/service/consul"
	"github.com/Comcast/webpa-common/service/zk"
)

// resetEnvironmentFactories resets the global factories for service.Environment objects
func resetEnvironmentFactories() {
	zookeeperEnvironmentFactory = zk.NewEnvironment
	consulEnvironmentFactory = consul.NewEnvironment
}

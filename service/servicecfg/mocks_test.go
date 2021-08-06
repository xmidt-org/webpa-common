package servicecfg

import (
	"github.com/xmidt-org/webpa-common/v2/service/consul"
	"github.com/xmidt-org/webpa-common/v2/service/zk"
)

// resetEnvironmentFactories resets the global factories for service.Environment objects
func resetEnvironmentFactories() {
	zookeeperEnvironmentFactory = zk.NewEnvironment
	consulEnvironmentFactory = consul.NewEnvironment
}

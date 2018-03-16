package deviceservice

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/monitor"
)

type ConnectorListener struct {
	Logger      log.Logger
	Environment service.Environment
	Connector   device.Connector
}

func (c *ConnectorListener) MonitorEvent(e monitor.Event) {
	logger := c.Logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	switch {
	case e.Err != nil:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery error", logging.ErrorKey(), e.Err)
		c.Connector.DisconnectAll()

	case e.Stopped:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery monitor being stopped")
		c.Connector.DisconnectAll()

	case len(e.Instances) > 0:
		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehashing devices", "instances", e.Instances)

		a := c.Environment.AccessorFactory()(e.Instances)
		disconnectCount := c.Connector.DisconnectIf(func(id device.ID) bool {
			instance, err := a.Get(id.Bytes())
			if err != nil {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting device: error during rehash", logging.ErrorKey(), err, "id", id)
				return true
			}

			if !c.Environment.IsRegistered(instance) {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting device: rehashed to another instance", "instance", instance, "id", id)
				return true
			}

			return true
		})

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash complete", "disconnectCount", disconnectCount)

	default:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery updated with no instances")
		c.Connector.DisconnectAll()
	}
}

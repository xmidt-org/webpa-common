package consul

import (
	"errors"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

//DatacenterWatcher checks if datacenters have been updated, based on an interval
//If they have, then the DatacenterWatcher will update the Instancers dictionary
type DatacenterWatcher struct {
	watchInterval time.Duration
	logger        log.Logger
	shutdown      chan struct{}
	environment   Environment
	options       Options
}

var (
	defaultLogger = log.NewNopLogger()
)

func newDatacenterWatcher(logger log.Logger, environment Environment, options Options) (*DatacenterWatcher, error) {
	if options.DatacenterWatchInterval == 0 {
		return nil, errors.New("interval cannot be 0")
	}

	if logger == nil {
		logger = defaultLogger
	}

	return &DatacenterWatcher{
		watchInterval: options.DatacenterWatchInterval,
		logger:        logger,
		shutdown:      make(chan struct{}),
		environment:   environment,
		options:       options,
	}, nil

}

func (i *DatacenterWatcher) Start() {
	go i.watchDatacenters()
}

func (i *DatacenterWatcher) Stop() {
	close(i.shutdown)
}

func (i *DatacenterWatcher) watchDatacenters() {

	environment := i.environment
	client := i.environment.Client()
	logger := i.logger
	options := i.options

	checkDatacenters := time.NewTicker(i.watchInterval)

	for {
		select {
		case <-i.shutdown:
			return
		case <-checkDatacenters.C:
			currentInstancers := environment.Instancers()
			keys := make(map[string]bool)
			instancersToAdd := make(service.Instancers)

			datacenters, err := getDatacenters(logger, client, options)

			if err != nil {
				continue
			}

			for _, w := range options.watches() {
				if w.CrossDatacenter {
					for _, datacenter := range datacenters {
						w.QueryOptions.Datacenter = datacenter

						// create keys for all datacenters + watched services
						key := newInstancerKey(w)
						keys[key] = true

						// don't create new instancer if it is already saved in environment's instancers
						if currentInstancers.Has(key) {
							continue
						}

						// don't create new instancer if it was already created and added to the new instancers map
						if instancersToAdd.Has(key) {
							logger.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly, "datacenter", w.QueryOptions.Datacenter)
							continue
						}

						// create new instancer and add it to the map of instancers to add
						instancersToAdd.Set(key, newInstancer(logger, client, w))
					}
				}
			}

			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "before instancers updated ", "oldInstancers: ", currentInstancers)
			environment.UpdateInstancers(keys, instancersToAdd)
			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "after instancers updated", "newInstancers: ", environment.Instancers())

		}

	}
}

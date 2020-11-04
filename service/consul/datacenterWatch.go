package consul

import (
	"errors"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

//datacenterWatcher checks if datacenters have been updated, based on an interval
//If they have, then the datacenterWatcher will update the Instancers dictionary
type datacenterWatcher struct {
	watchInterval time.Duration
	logger        log.Logger
	shutdown      chan struct{}
	environment   Environment
	options       Options
}

var (
	defaultLogger = log.NewNopLogger()
)

func newDatacenterWatcher(logger log.Logger, environment Environment, options Options, chrysomConfig chrysom.ClientConfig) (*datacenterWatcher, error) {
	var chrysomClient *chrysom.Client

	if options.DatacenterWatchInterval == 0 {
		return nil, errors.New("interval cannot be 0")
	}

	if logger == nil {
		logger = defaultLogger
	}

	if chrysomConfig.PullInterval > 0 {
		chrysomClient, err := chrysom.CreateClient(chrysomConfig)

		if err != nil {
			return nil, errors.New("could not create chrysom client")
		}
	}

	return &datacenterWatcher{
		watchInterval: options.DatacenterWatchInterval,
		logger:        logger,
		shutdown:      make(chan struct{}),
		environment:   environment,
		options:       options,
		chrysomClient: chrysomClient,
	}, nil

}

func (i *datacenterWatcher) start() {
	go i.watchDatacenters()
}

func (i *datacenterWatcher) stop() {
	close(i.shutdown)
}

// TODO: add a function to start the chrysom client ticker

func (i *datacenterWatcher) watchDatacenters() {

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

			keys, instancersToAdd := i.datacenterInstancersUpdate(datacenters, currentInstancers)
			environment.UpdateInstancers(keys, instancersToAdd)

		}

	}
}

func (i *datacenterWatcher) datacenterInstancersUpdate(datacenters []string, currentInstancers service.Instancers) (map[string]bool, service.Instancers) {
	keys := make(map[string]bool)
	instancersToAdd := make(service.Instancers)

	options := i.options
	client := i.environment.Client()
	logger := i.logger

	for _, w := range options.watches() {
		if w.CrossDatacenter {
			for _, datacenter := range datacenters {
				// TODO: check if it's in the inactive datacenters list first
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
			}

			environment.UpdateInstancers(keys, instancersToAdd)

		}

	}
}

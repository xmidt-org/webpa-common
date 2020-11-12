package consul

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/mapstructure"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

//DatacenterWatcher checks if datacenters have been updated, based on an interval
type datacenterWatcher struct {
	logger                log.Logger
	environment           Environment
	options               Options
	inactiveDatacenters   map[string]bool
	chrysomClient         *chrysom.Client
	consulDatacenterWatch *consulDatacenterWatch
	lock                  sync.RWMutex
}

type consulDatacenterWatch struct {
	watchInterval time.Duration
	shutdown      chan struct{}
}

type datacenterFilter struct {
	Name     string `mapstructure:"name"`
	Inactive bool   `mapstructure:"inactive"`
}

var (
	defaultLogger = log.NewNopLogger()
)

func NewDatacenterWatcher(logger log.Logger, environment Environment, options Options) (*datacenterWatcher, error) {
	var consulWatch *consulDatacenterWatch

	if logger == nil {
		logger = defaultLogger
	}

	if options.DatacenterWatchInterval <= 0 {
		//default consul interval is 5m
		options.DatacenterWatchInterval = time.Duration(5 * time.Minute)
	}

	consulWatch = &consulDatacenterWatch{
		watchInterval: options.DatacenterWatchInterval,
		shutdown:      make(chan struct{}),
	}

	datacenterWatcher := &datacenterWatcher{
		consulDatacenterWatch: consulWatch,
		logger:                logger,
		options:               options,
		environment:           environment,
		inactiveDatacenters:   make(map[string]bool),
	}

	if options.ChrysomConfig != nil && options.ChrysomConfig.PullInterval > 0 {

		if environment.Provider() == nil {
			return nil, errors.New("must pass in a metrics provider")
		}

		options.ChrysomConfig.MetricsProvider = environment.Provider()

		var datacenterListenerFunc chrysom.ListenerFunc = func(items []model.Item) {
			updateInactiveDatacenters(items, datacenterWatcher.inactiveDatacenters, &datacenterWatcher.lock, logger)
		}

		options.ChrysomConfig.Listener = datacenterListenerFunc

		options.ChrysomConfig.Logger = logger
		chrysomClient, err := chrysom.CreateClient(*options.ChrysomConfig)

		if err != nil {
			return nil, err
		}

		//create chrysom client and start it
		datacenterWatcher.chrysomClient = chrysomClient
		datacenterWatcher.chrysomClient.Start(context.Background())

	} else if options.ChrysomConfig != nil && options.ChrysomConfig.PullInterval <= 0 {
		return nil, errors.New("chrysom pull interval cannot be 0")
	}

	//start consul watch
	ticker := time.NewTicker(datacenterWatcher.consulDatacenterWatch.watchInterval)
	go datacenterWatcher.watchDatacenters(ticker)

	return datacenterWatcher, nil

}

func (d *datacenterWatcher) Stop() {
	close(d.consulDatacenterWatch.shutdown)

	if d.chrysomClient != nil {
		d.chrysomClient.Stop(context.Background())
	}
}

func (d *datacenterWatcher) watchDatacenters(ticker *time.Ticker) {
	for {
		select {
		case <-d.consulDatacenterWatch.shutdown:
			ticker.Stop()
			return
		case <-d.environment.Closed():
			ticker.Stop()
			d.Stop()
			return
		case <-ticker.C:
			datacenters, err := getDatacenters(d.logger, d.environment.Client(), d.options)

			if err != nil {
				continue
			}

			d.UpdateInstancers(datacenters)

		}

	}
}

func (d *datacenterWatcher) UpdateInstancers(datacenters []string) {
	keys := make(map[string]bool)
	instancersToAdd := make(service.Instancers)

	currentInstancers := d.environment.Instancers()

	for _, w := range d.options.watches() {
		if w.CrossDatacenter {
			for _, datacenter := range datacenters {

				//check if datacenter is part of inactive datacenters list
				d.lock.RLock()
				_, found := d.inactiveDatacenters[datacenter]
				d.lock.RUnlock()

				if found {
					d.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "datacenter set as inactive", "datacenter name: ", datacenter)
					continue
				}
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
					d.logger.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly, "datacenter", w.QueryOptions.Datacenter)
					continue
				}

				// create new instancer and add it to the map of instancers to add
				instancersToAdd.Set(key, newInstancer(d.logger, d.environment.Client(), w))
			}
		}
	}

	d.environment.UpdateInstancers(keys, instancersToAdd)

}

func updateInactiveDatacenters(items []model.Item, inactiveDatacenters map[string]bool, lock *sync.RWMutex, logger log.Logger) {
	chrysomMap := make(map[string]bool)
	for _, item := range items {

		var df datacenterFilter

		// decode database item data into datacenter filter structure
		err := mapstructure.Decode(item.Data, &df)

		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "failed to decode database results into datacenter filter struct")
			continue
		}

		if df.Inactive {
			chrysomMap[df.Name] = true
			lock.Lock()
			inactiveDatacenters[df.Name] = true
			lock.Unlock()
		}
	}

	lock.Lock()

	for key := range inactiveDatacenters {
		_, ok := chrysomMap[key]

		if !ok {
			delete(inactiveDatacenters, key)
		}
	}

	lock.Unlock()
}

package consul

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

//datacenterWatcher checks if datacenters have been updated, based on an interval
type DatacenterWatcher struct {
	logger                 log.Logger
	environment            Environment
	options                Options
	inactiveDatacenters    map[string]bool
	chrysomDatacenterWatch *chrysomDatacenterWatch
	consulDatacenterWatch  *consulDatacenterWatch
	lock                   sync.RWMutex
}

type chrysomDatacenterWatch struct {
	chrysomClient *chrysom.Client
	ctx           context.Context
}

type consulDatacenterWatch struct {
	watchInterval time.Duration
	shutdown      chan struct{}
}

var (
	defaultLogger = log.NewNopLogger()
)

func NewDatacenterWatcher(logger log.Logger, environment Environment, options Options, ctx context.Context) (*DatacenterWatcher, error) {
	var consulWatch *consulDatacenterWatch

	if logger == nil {
		logger = defaultLogger
	}

	if options.DatacenterWatchInterval > 0 {
		consulWatch = &consulDatacenterWatch{
			watchInterval: options.DatacenterWatchInterval,
			shutdown:      make(chan struct{}),
		}
	}

	datacenterWatcher := &DatacenterWatcher{
		consulDatacenterWatch: consulWatch,
		logger:                logger,
		options:               options,
		environment:           environment,
		inactiveDatacenters:   make(map[string]bool),
	}

	if options.ChrysomConfig.PullInterval > 0 {

		if environment.Provider() == nil {
			return nil, errors.New("must pass in a metrics provider")
		}

		options.ChrysomConfig.MetricsProvider = environment.Provider()
		options.ChrysomConfig.Listener = datacenterWatcher.DatacentersListener()
		options.ChrysomConfig.Logger = logger
		chrysomClient, err := chrysom.CreateClient(*options.ChrysomConfig)

		if err != nil {
			return nil, err
		}

		if ctx == nil {
			ctx = context.Background()
		}

		datacenterWatcher.chrysomDatacenterWatch = &chrysomDatacenterWatch{
			chrysomClient: chrysomClient,
			ctx:           ctx,
		}
	}

	return datacenterWatcher, nil

}

func (d *DatacenterWatcher) StartConsulTicker() {
	if d.consulDatacenterWatch.watchInterval > 0 {
		ticker := time.NewTicker(d.consulDatacenterWatch.watchInterval)
		go d.watchDatacenters(ticker)
	}
}

func (d *DatacenterWatcher) StopConsulTicker() {
	close(d.consulDatacenterWatch.shutdown)
}

func (d *DatacenterWatcher) StartChrysomTicker() {
	if d.chrysomDatacenterWatch != nil {
		d.chrysomDatacenterWatch.chrysomClient.Start(d.chrysomDatacenterWatch.ctx)
	}
}

func (d *DatacenterWatcher) StopChrysomTicker() {
	if d.chrysomDatacenterWatch != nil {
		d.chrysomDatacenterWatch.chrysomClient.Stop(d.chrysomDatacenterWatch.ctx)
	}
}

func (d *DatacenterWatcher) watchDatacenters(ticker *time.Ticker) {

	client := d.environment.Client()
	logger := d.logger
	options := d.options

	for {
		select {
		case <-d.consulDatacenterWatch.shutdown:
			return
		case <-ticker.C:
			d.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "consul timer")

			datacenters, err := getDatacenters(logger, client, options)

			if err != nil {
				continue
			}

			d.UpdateInstancers(datacenters)

		}

	}
}

func (d *DatacenterWatcher) UpdateInstancers(datacenters []string) {
	keys := make(map[string]bool)
	instancersToAdd := make(service.Instancers)

	options := d.options
	environment := d.environment
	logger := d.logger
	currentInstancers := environment.Instancers()

	for _, w := range options.watches() {
		if w.CrossDatacenter {
			for _, datacenter := range datacenters {

				//check if datacenter is part of inactive datacenters list
				d.lock.RLock()
				_, found := d.inactiveDatacenters[datacenter]
				d.lock.RUnlock()

				if found {
					logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "datacenter set as inactive", "datacenter name: ", datacenter)
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
					logger.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "skipping duplicate watch", "service", w.Service, "tags", w.Tags, "passingOnly", w.PassingOnly, "datacenter", w.QueryOptions.Datacenter)
					continue
				}

				// create new instancer and add it to the map of instancers to add
				instancersToAdd.Set(key, newInstancer(logger, d.environment.Client(), w))
			}
		}
	}

	d.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "BEFORE instancers update", "instancers: ", environment.Instancers())

	environment.UpdateInstancers(keys, instancersToAdd)

	d.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "AFTER instancers update", "instancers: ", environment.Instancers())

}

func (d *DatacenterWatcher) DatacentersListener() chrysom.ListenerFunc {
	return func(items []model.Item) {
		d.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "getting from chrysom database", "items: ", items)
		for _, item := range items {
			datacenterName := item.Data["name"].(string)
			if item.Data["active"] == true {
				d.lock.Lock()
				delete(d.inactiveDatacenters, datacenterName)
				d.lock.Unlock()
			} else {
				d.lock.Lock()
				d.inactiveDatacenters[datacenterName] = true
				d.lock.Unlock()
			}
		}

		datacenters, err := getDatacenters(d.logger, d.environment.Client(), d.options)

		if err == nil {
			d.UpdateInstancers(datacenters)
		}

	}
}

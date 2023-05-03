package consul

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/service"
	"go.uber.org/zap"
)

// datacenterWatcher checks if datacenters have been updated, based on an interval.
type datacenterWatcher struct {
	logger              *zap.Logger
	environment         Environment
	options             Options
	inactiveDatacenters map[string]bool
	stopListener        func(context.Context) error
	consulWatchInterval time.Duration
	lock                sync.RWMutex
}

type datacenterFilter struct {
	Name     string
	Inactive bool
}

var (
	defaultWatchInterval = 5 * time.Minute
)

func newDatacenterWatcher(logger *zap.Logger, environment Environment, options Options) (*datacenterWatcher, error) {

	if logger == nil {
		logger = sallust.Default()
	}

	if options.DatacenterWatchInterval <= 0 {
		//default consul interval is 5m
		options.DatacenterWatchInterval = defaultWatchInterval
	}

	datacenterWatcher := &datacenterWatcher{
		consulWatchInterval: options.DatacenterWatchInterval,
		logger:              logger,
		options:             options,
		environment:         environment,
		inactiveDatacenters: make(map[string]bool),
	}

	if len(options.Chrysom.Bucket) > 0 {
		if options.Chrysom.Listen.PullInterval <= 0 {
			return nil, errors.New("chrysom pull interval cannot be 0")
		}

		// only chrysom client uses the provider for metrics
		if environment.Provider() == nil {
			return nil, errors.New("must pass in a metrics provider")
		}

		var datacenterListenerFunc chrysom.ListenerFunc = func(items chrysom.Items) {
			updateInactiveDatacenters(items, datacenterWatcher.inactiveDatacenters, &datacenterWatcher.lock, logger)
		}

		options.Chrysom.Listen.Listener = datacenterListenerFunc

		m := &chrysom.Measures{
			Polls: environment.Provider().NewCounterVec(chrysom.PollCounter),
		}
		basic, err := chrysom.NewBasicClient(options.Chrysom.BasicClientConfig, sallust.Get)
		if err != nil {
			return nil, fmt.Errorf("failed to create chrysom basic client: %v", err)
		}
		listener, err := chrysom.NewListenerClient(options.Chrysom.Listen, sallust.With, m, basic)
		if err != nil {
			return nil, fmt.Errorf("failed to create chrysom listener client: %v", err)
		}

		//create chrysom client and start it
		datacenterWatcher.stopListener = listener.Stop
		listener.Start(context.Background())
		logger.Debug("started chrysom, argus client")
	}

	//start consul watch
	ticker := time.NewTicker(datacenterWatcher.consulWatchInterval)
	go datacenterWatcher.watchDatacenters(ticker)
	logger.Debug("started consul datacenter watch")
	return datacenterWatcher, nil

}

func (d *datacenterWatcher) stop() {
	if d.stopListener != nil {
		d.stopListener(context.Background())
	}
}

func (d *datacenterWatcher) watchDatacenters(ticker *time.Ticker) {
	for {
		select {
		case <-d.environment.Closed():
			ticker.Stop()
			d.stop()
			return
		case <-ticker.C:
			datacenters, err := getDatacenters(d.logger, d.environment.Client(), d.options)

			if err != nil {
				// getDatacenters function logs the error, but a metric should be added
				continue
			}

			d.updateInstancers(datacenters)

		}

	}
}

func (d *datacenterWatcher) updateInstancers(datacenters []string) {
	keys := make(map[string]bool)
	instancersToAdd := make(service.Instancers)

	currentInstancers := d.environment.Instancers()

	for _, w := range d.options.watches() {
		if w.CrossDatacenter {
			for _, datacenter := range datacenters {

				createNewInstancer(keys, instancersToAdd, currentInstancers, d, datacenter, w)
			}
		}
	}

	d.environment.UpdateInstancers(keys, instancersToAdd)

}

func updateInactiveDatacenters(items []model.Item, inactiveDatacenters map[string]bool, lock *sync.RWMutex, logger *zap.Logger) {
	chrysomMap := make(map[string]bool)
	for _, item := range items {

		var df datacenterFilter

		// decode database item data into datacenter filter structure
		err := mapstructure.Decode(item.Data, &df)

		if err != nil {
			logger.Error("failed to decode database results into datacenter filter struct")
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

func createNewInstancer(keys map[string]bool, instancersToAdd service.Instancers, currentInstancers service.Instancers, dw *datacenterWatcher, datacenter string, w Watch) {
	//check if datacenter is part of inactive datacenters list
	dw.lock.RLock()
	_, found := dw.inactiveDatacenters[datacenter]
	dw.lock.RUnlock()

	if found {
		field := zap.Any("datacenter name: ", datacenter)
		dw.logger.Info("datacenter set as inactive", field)
		return
	}

	w.QueryOptions.Datacenter = datacenter

	// create keys for all datacenters + watched services
	key := newInstancerKey(w)
	keys[key] = true

	// don't create new instancer if it is already saved in environment's instancers
	if currentInstancers.Has(key) {
		return
	}

	// don't create new instancer if it was already created and added to the new instancers map
	if instancersToAdd.Has(key) {
		s := zap.String("service", w.Service)
		t := zap.Any("tags", w.Tags)
		p := zap.Bool("passingOnly", w.PassingOnly)
		d := zap.String("datacenter", w.QueryOptions.Datacenter)
		dw.logger.Warn("skipping duplicate watch", s, t, p, d)
		return
	}

	// create new instancer and add it to the map of instancers to add
	instancersToAdd.Set(key, newInstancer(dw.logger, dw.environment.Client(), w))
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"time"

	"github.com/xmidt-org/webpa-common/v2/adapter"
	"github.com/xmidt-org/webpa-common/v2/service"
	"go.uber.org/zap"
)

// datacenterWatcher checks if datacenters have been updated, based on an interval.
type datacenterWatcher struct {
	logger              *zap.Logger
	environment         Environment
	options             Options
	consulWatchInterval time.Duration
}

var (
	defaultWatchInterval = 5 * time.Minute
)

func newDatacenterWatcher(logger *zap.Logger, environment Environment, options Options) (*datacenterWatcher, error) {

	if logger == nil {
		logger = adapter.DefaultLogger().Logger
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
	}

	//start consul watch
	ticker := time.NewTicker(datacenterWatcher.consulWatchInterval)
	go datacenterWatcher.watchDatacenters(ticker)
	logger.Debug("started consul datacenter watch")
	return datacenterWatcher, nil

}

func (d *datacenterWatcher) watchDatacenters(ticker *time.Ticker) {
	for {
		select {
		case <-d.environment.Closed():
			ticker.Stop()
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

func createNewInstancer(keys map[string]bool, instancersToAdd service.Instancers, currentInstancers service.Instancers, dw *datacenterWatcher, datacenter string, w Watch) {
	//check if datacenter is part of inactive datacenters list
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
		dw.logger.Warn("skipping duplicate watch", zap.String("service", w.Service), zap.Any("tags", w.Tags), zap.Bool("passingOnly", w.PassingOnly), zap.String("datacenter", w.QueryOptions.Datacenter))
		return
	}

	// create new instancer and add it to the map of instancers to add
	instancersToAdd.Set(key, newInstancer(dw.logger, dw.environment.Client(), w))
}

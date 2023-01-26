package consul

import (
	"context"
	"errors"
	"net/url"
	"regexp"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/service/monitor"
	"github.com/xmidt-org/webpa-common/v2/xresolver"
	"go.uber.org/zap"
)

var find = regexp.MustCompile("(.*)" + regexp.QuoteMeta("[") + "(.*)" + regexp.QuoteMeta("]") + regexp.QuoteMeta("{") + "(.*)" + regexp.QuoteMeta("}"))

type Options struct {
	// Watch is what to url to match with the consul service
	// exp. { "http://beta.google.com:8080/notify" : "caduceus" }
	Watch map[string]string `json:"watch"`

	Logger *zap.Logger `json:"-"`
}

type ConsulWatcher struct {
	logger *zap.Logger

	watch     map[string]string
	balancers map[string]*xresolver.RoundRobin
}

func NewConsulWatcher(o Options) *ConsulWatcher {
	if o.Logger == nil {
		o.Logger = sallust.Default()
	}

	watcher := &ConsulWatcher{
		logger:    o.Logger.With(zap.String("component", "consulWatcher")),
		balancers: make(map[string]*xresolver.RoundRobin),
		watch:     make(map[string]string),
	}

	if o.Watch != nil {
		for url, service := range o.Watch {
			watcher.WatchService(url, service)
		}
	}

	return watcher
}

func (watcher *ConsulWatcher) MonitorEvent(e monitor.Event) {
	watcher.logger.Debug("received update route event", zap.Any("event", e))

	// update balancers
	str := find.FindStringSubmatch(e.Key)
	if len(str) < 3 {
		return
	}

	service := str[1]
	if rr, found := watcher.balancers[service]; found {
		routes := make([]xresolver.Route, 0)
		for _, instance := range e.Instances {
			// find records
			route, err := xresolver.CreateRoute(instance)
			if err != nil {
				watcher.logger.Error("failed to create route", zap.Error(err), zap.String("instance", instance))
				continue
			}
			routes = append(routes, route)
		}
		rr.Update(routes)
		watcher.logger.Info("updating routes", zap.String("service", service), zap.Any("new-routes", routes))
	}
}

func (watcher *ConsulWatcher) WatchService(watchURL string, service string) {
	parsedURL, err := url.Parse(watchURL)
	if err != nil {
		watcher.logger.Error("failed to parse url", zap.String("url", watchURL), zap.String("service", service))
		return
	}
	watcher.logger.Info("Watching Service", zap.String("url", watchURL), zap.String("service", service), zap.String("host", parsedURL.Hostname()))

	if _, found := watcher.watch[parsedURL.Hostname()]; !found {
		watcher.watch[parsedURL.Hostname()] = service
		if _, found := watcher.balancers[service]; !found {
			watcher.balancers[service] = xresolver.NewRoundRobinBalancer()
		}
	}
}

func (watcher *ConsulWatcher) LookupRoutes(ctx context.Context, host string) ([]xresolver.Route, error) {
	if _, found := watcher.watch[host]; !found {
		watcher.logger.Error("watch not found ", zap.String("host", host))
		return []xresolver.Route{}, errors.New(host + " is not part of the consul listener")
	}
	records, err := watcher.balancers[watcher.watch[host]].Get()
	watcher.logger.Debug("looking up routes", zap.Any("routes", records), zap.Error(err))
	return records, err
}

package consul

import (
	"context"
	"errors"
	"net/url"
	"regexp"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/service/monitor"
	"github.com/xmidt-org/webpa-common/v2/xresolver"
)

var find = regexp.MustCompile("(.*)" + regexp.QuoteMeta("[") + "(.*)" + regexp.QuoteMeta("]") + regexp.QuoteMeta("{") + "(.*)" + regexp.QuoteMeta("}"))

type Options struct {
	// Watch is what to url to match with the consul service
	// exp. { "http://beta.google.com:8080/notify" : "caduceus" }
	Watch map[string]string `json:"watch"`

	Logger log.Logger `json:"-"`
}

type ConsulWatcher struct {
	logger log.Logger

	watch     map[string]string
	balancers map[string]*xresolver.RoundRobin
}

func NewConsulWatcher(o Options) *ConsulWatcher {
	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	watcher := &ConsulWatcher{
		logger:    log.WithPrefix(o.Logger, "component", "consulWatcher"),
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
	log.WithPrefix(watcher.logger, level.Key(), level.DebugValue()).Log(logging.MessageKey(), "received update route event", "event", e)

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
				log.WithPrefix(watcher.logger, level.Key(), level.ErrorValue()).Log(logging.MessageKey(), "failed to create route", logging.MessageKey(), err, "instance", instance)
				continue
			}
			routes = append(routes, route)
		}
		rr.Update(routes)
		log.WithPrefix(watcher.logger, level.Key(), level.InfoValue()).Log(logging.MessageKey(), "updating routes", "service", service, "new-routes", routes)
	}
}

func (watcher *ConsulWatcher) WatchService(watchURL string, service string) {
	parsedURL, err := url.Parse(watchURL)
	if err != nil {
		log.WithPrefix(watcher.logger, level.Key(), level.ErrorValue()).Log("failed to parse url", "url", watchURL, "service", service)
		return
	}
	log.WithPrefix(watcher.logger, level.Key(), level.InfoValue()).Log(logging.MessageKey(), "Watching Service", "url", watchURL, "service", service, "host", parsedURL.Hostname())

	if _, found := watcher.watch[parsedURL.Hostname()]; !found {
		watcher.watch[parsedURL.Hostname()] = service
		if _, found := watcher.balancers[service]; !found {
			watcher.balancers[service] = xresolver.NewRoundRobinBalancer()
		}
	}
}

func (watcher *ConsulWatcher) LookupRoutes(ctx context.Context, host string) ([]xresolver.Route, error) {
	if _, found := watcher.watch[host]; !found {
		log.WithPrefix(watcher.logger, level.Key(), level.ErrorValue()).Log("watch not found ", "host", host)
		return []xresolver.Route{}, errors.New(host + " is not part of the consul listener")
	}
	records, err := watcher.balancers[watcher.watch[host]].Get()
	log.WithPrefix(watcher.logger, level.Key(), level.DebugValue()).Log(logging.MessageKey(), "looking up routes", "routes", records, logging.ErrorKey(), err)
	return records, err
}

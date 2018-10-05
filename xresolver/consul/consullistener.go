package consul

import (
	"context"
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service/monitor"
	"github.com/Comcast/webpa-common/xresolver"
	"github.com/go-kit/kit/log"

	"regexp"
)

var find = regexp.MustCompile("(.*)" + regexp.QuoteMeta("[") + "(.*)" + regexp.QuoteMeta("]") + regexp.QuoteMeta("{") + "(.*)" + regexp.QuoteMeta("}"))

type Options struct {
	// Watch is what to url to match with the consul service
	// exp. { "beta.google.com" : "caduceus" }
	Watch map[string]string `json:"watch"`

	Logger log.Logger `json:"-"`
}

type ConsulWatcher struct {
	debugLogger log.Logger
	infoLogger  log.Logger
	warnLogger  log.Logger
	errorLogger log.Logger

	balancers map[string]*xresolver.RoundRobin
	config    *Options
}

func NewConsulWatcher(o *Options) *ConsulWatcher {
	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	balancers := make(map[string]*xresolver.RoundRobin)
	for _, service := range o.Watch {
		if _, found := balancers[service]; !found {
			balancers[service] = xresolver.NewRoundRobinBalancer()
		}
	}
	watcher := &ConsulWatcher{
		balancers:   balancers,
		config:      o,
		debugLogger: logging.Debug(o.Logger),
		infoLogger:  logging.Info(o.Logger),
		warnLogger:  logging.Warn(o.Logger),
		errorLogger: logging.Error(o.Logger),
	}

	return watcher
}

func (watcher *ConsulWatcher) MonitorEvent(e monitor.Event) {
	// update balancers
	str := find.FindStringSubmatch(e.Key)
	if len(str) < 3 {
		return
	}
	if rr, found := watcher.balancers[str[1]]; found {
		routes := make([]xresolver.Route, len(e.Instances))
		for index, instance := range e.Instances {
			// find records
			route, err := xresolver.CreateRoute(instance)
			if err != nil {
				watcher.errorLogger.Log(logging.MessageKey(), "failed to create route", logging.MessageKey(), err, "instance", instance)
				continue
			}
			routes[index] = *route
		}
		rr.Update(routes)
		watcher.debugLogger.Log(logging.MessageKey(), "received update event", "service", str[1], "new-routes", routes)
	}
}

func (watcher *ConsulWatcher) LookupRoutes(ctx context.Context, host string) ([]xresolver.Route, error) {
	if _, found := watcher.config.Watch[host]; !found {
		return []xresolver.Route{}, errors.New(host + " is not part of the consul listener")
	}
	records, err := watcher.balancers[watcher.config.Watch[host]].Get()
	watcher.debugLogger.Log(logging.MessageKey(), "looking up routes", "routes", records, logging.ErrorKey(), err)
	return records, err
}

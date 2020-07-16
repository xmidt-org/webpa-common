package xresolver

import (
	"context"
	"errors"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"

	"net"
	"strconv"
	"sync"
)

// Note to self: Dial is not being set for net.Resolver because that is the Dial to the DNS server.

var DefaultDialer = net.Dialer{}

type resolver struct {
	resolvers map[Lookup]bool
	lock      sync.RWMutex
	dialer    net.Dialer
	logger    log.Logger
}

func NewResolver(dialer net.Dialer, logger log.Logger, lookups ...Lookup) Resolver {
	if logger == nil {
		logger = logging.DefaultLogger()
	}
	r := &resolver{
		resolvers: make(map[Lookup]bool),
		dialer:    dialer,
		logger:    log.WithPrefix(logger, "component", "xresolver"),
	}

	for _, lookup := range lookups {
		r.Add(lookup)
	}
	return r
}

func (resolve *resolver) Add(r Lookup) error {
	resolve.lock.RLock()
	found := resolve.resolvers[r]
	resolve.lock.RUnlock()
	if found {
		return errors.New("resolver already exist")
	}

	resolve.lock.Lock()
	resolve.resolvers[r] = true
	resolve.lock.Unlock()
	return nil
}

func (resolve *resolver) Remove(r Lookup) error {
	resolve.lock.RLock()
	found := resolve.resolvers[r]
	resolve.lock.RUnlock()
	if !found {
		return errors.New("resolver does not exist")
	}

	resolve.lock.Lock()
	delete(resolve.resolvers, r)
	resolve.lock.Unlock()
	return nil
}

func (resolve *resolver) getRoutes(ctx context.Context, host string) []Route {
	routes := make([]Route, 0)
	for r := range resolve.resolvers {
		tempRoutes, err := r.LookupRoutes(ctx, host)
		if err == nil {
			routes = append(routes, tempRoutes...)
		}
	}

	return routes
}

func (resolve *resolver) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return resolve.dialer.Dial(network, net.JoinHostPort(ip.String(), port))
	}

	// get records using custom resolvers
	routes := resolve.getRoutes(ctx, host)

	// generate Conn or err from records
	con, route, err := resolve.createConnection(routes, network, port)
	if err == nil {
		log.WithPrefix(resolve.logger, level.Key(), level.DebugValue()).Log(logging.MessageKey(), "successfully created connection using xresolver", "new-route", route.String(), "addr", addr)
		return con, err
	}

	log.WithPrefix(resolve.logger, level.Key(), level.DebugValue()).Log(logging.MessageKey(), "failed to create connection with other routes using original address", "addr", addr, logging.ErrorKey(), err)
	// if no connection, create using the default dialer
	return resolve.dialer.DialContext(ctx, network, addr)
}

func (resolve *resolver) createConnection(routes []Route, network, port string) (net.Conn, Route, error) {
	for _, route := range routes {
		portUsed := port
		if route.Port != 0 {
			portUsed = strconv.Itoa(route.Port)
		} else {
			if route.Scheme == "http" {
				portUsed = "80"
			} else if route.Scheme == "https" {
				portUsed = "443"
			} else {
				log.WithPrefix(resolve.logger, level.Key(), level.ErrorValue()).Log(logging.MessageKey(), "failed to create default port", "scheme", route.Scheme, "host", route.Host)
			}

		}
		con, err := resolve.dialer.Dial(network, net.JoinHostPort(route.Host, portUsed))
		if err == nil {
			return con, route, err
		}
	}
	return nil, Route{}, errors.New("failed to create connection from routes")
}

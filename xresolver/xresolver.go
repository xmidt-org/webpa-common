package xresolver

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
)

// Note to self: Dial is not being set for net.Resolver because that is the Dial to the DNS server.

var DefaultDialer = &net.Dialer{}

type resolver struct {
	resolvers map[Lookup]struct{}
	lock      *sync.RWMutex
	dialer    *net.Dialer
}

func NewResolver(dialer *net.Dialer, lookups ...Lookup) Resolver {
	if dialer == nil {
		dialer = DefaultDialer
	}
	r := &resolver{
		resolvers: make(map[Lookup]struct{}),
		lock:      new(sync.RWMutex),
		dialer:    dialer,
	}

	for _, lookup := range lookups {
		r.Add(lookup)
	}

	return r
}

func (resolve *resolver) Add(r Lookup) error {
	resolve.lock.RLock()
	_, found := resolve.resolvers[r]
	resolve.lock.RUnlock()
	if found {
		return errors.New("resolver already exist")
	}

	resolve.lock.Lock()
	defer resolve.lock.Unlock()
	resolve.resolvers[r] = struct{}{}
	return nil
}

func (resolve *resolver) Remove(r Lookup) error {
	resolve.lock.RLock()
	_, found := resolve.resolvers[r]
	resolve.lock.RUnlock()
	if !found {
		return errors.New("resolver does not exist")
	}

	resolve.lock.Lock()
	defer resolve.lock.Unlock()
	delete(resolve.resolvers, r)
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

func (resolve *resolver) DialContext(ctx context.Context, network, addr string) (con net.Conn, err error) {
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
	con, err = resolve.createConnection(routes, network, port)
	if err == nil {
		return
	}

	// if no connection, create using the default dialer
	return resolve.dialer.DialContext(ctx, network, addr)
}

func (resolve *resolver) createConnection(routes []Route, network, port string) (con net.Conn, err error) {
	for _, route := range routes {
		portUsed := port
		if route.Port != 0 {
			portUsed = strconv.Itoa(route.Port)
		}
		con, err = resolve.dialer.Dial(network, net.JoinHostPort(route.Host, portUsed))
		if err == nil {
			return
		}
	}
	return nil, errors.New("failed to create connection from routes")
}

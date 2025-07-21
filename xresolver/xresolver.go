// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xresolver

import (
	"context"
	"errors"

	"github.com/xmidt-org/sallust"
	"go.uber.org/zap"

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
	logger    *zap.Logger
}

func NewResolver(dialer net.Dialer, logger *zap.Logger, lookups ...Lookup) Resolver {
	if logger == nil {
		logger = sallust.Default()
	}
	r := &resolver{
		resolvers: make(map[Lookup]bool),
		dialer:    dialer,
		logger:    logger.With(zap.String("component", "xresolver")),
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
		resolve.logger.Debug("successfully created connection using xresolver", zap.String("new-route", route.String()), zap.String("addr", addr))
		return con, err
	}

	resolve.logger.Debug("failed to create connection with other routes using original address", zap.String("addr", addr), zap.Error(err))
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
				resolve.logger.Error("unknown default port", zap.String("scheme", route.Scheme), zap.String("host", route.Host))
				continue
			}
		}
		con, err := resolve.dialer.Dial(network, net.JoinHostPort(route.Host, portUsed))
		if err == nil {
			return con, route, err
		}
	}
	return nil, Route{}, errors.New("failed to create connection from routes")
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xresolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Lookup interface {
	// LookupIPAddr looks up host using the local resolver. It returns a slice of that host's IPv4 and IPv6 addresses.
	LookupRoutes(ctx context.Context, host string) ([]Route, error)
}

type Dial interface {
	// DialContext connects to the address on the named network using the provided context.
	DialContext(ctx context.Context, network, addr string) (con net.Conn, err error)
}

type ConnCreation interface {
	Dial(network, address string) (net.Conn, error)
}

// Resolver represents how to generate the address and how to create the connection
type Resolver interface {
	Dial

	// Add adds the resolver to the methods of creating the IPv4 and IPv6 addresses
	Add(r Lookup) error

	// Remove removes the resolver to the methods of creating the IPv4 and IPv6 addresses
	Remove(r Lookup) error
}

type Route struct {
	Scheme string
	Host   string
	Port   int
}

// instancePattern is what NormalizeInstance expects to be matched.  This pattern is intentionally liberal, and allows
// URIs that are disallowed under https://www.ietf.org/rfc/rfc2396.txt
var instancePattern = regexp.MustCompile("^((?P<scheme>.+)://)?(?P<address>[^:]+)(:(?P<port>[0-9]+))?$")

// NormalizeRoute canonicalizes a route string from a backend
//
// This function performs the following on the instance:
//   (1) If instance is a blank string, e.g. contains only whitespace or is empty, an empty string is returned with an error
//   (2) If the instance with whitespace trimmed off is not a valid instance, an error is returned with the trimmed instance string.
//       This function is intentionally lenient on what is a valid instance string, e.g. "foobar.com", "foobar.com:8080", "asdf://foobar.com", etc
//   (3) If there was no scheme prepended to the route, http will be used
func NormalizeRoute(route string) (string, error) {
	route = strings.TrimSpace(route)
	if len(route) == 0 {
		return route, errors.New("empty route is not allowed")
	}

	submatches := instancePattern.FindStringSubmatch(route)
	if len(submatches) == 0 {
		return route, fmt.Errorf("invalid route: %s", route)
	}

	var (
		scheme  = submatches[2]
		address = submatches[3]
	)

	if len(scheme) == 0 {
		scheme = "http"
	}

	var port int
	if portValue := submatches[5]; len(portValue) > 0 {
		var err error
		port, err = strconv.Atoi(submatches[5])
		if err != nil {
			// NOTE: Shouldn't ever hit this case, because the port is constrained by the regexp to be numeric
			return route, err
		}
		return fmt.Sprintf("%s://%s:%d", scheme, address, port), nil

	}
	return fmt.Sprintf("%s://%s", scheme, address), nil
}

func CreateRoute(route string) (Route, error) {
	route, err := NormalizeRoute(route)
	if err != nil {
		return Route{}, err
	}
	path, err := url.Parse(route)
	if err != nil {
		return Route{}, err
	}

	newRoute := Route{
		Scheme: path.Scheme,
		Host:   path.Hostname(),
	}
	if path.Port() != "" {
		port, err := strconv.Atoi(path.Port())
		newRoute.Port = port
		if err != nil {
			return newRoute, err
		}
	}
	return newRoute, nil
}

func (r Route) String() string {
	str := r.Scheme + "://" + r.Host
	if r.Port != 0 {
		return str + ":" + strconv.Itoa(r.Port)
	}
	return str
}

type orderedRoute struct {
	route Route
	index int
}

type RoundRobin struct {
	lock   sync.RWMutex
	routes map[string]*orderedRoute
}

func NewRoundRobinBalancer() *RoundRobin {
	return &RoundRobin{
		routes: make(map[string]*orderedRoute),
	}
}

func (robin *RoundRobin) Add(route Route) error {
	// check if exist
	robin.lock.RLock()
	_, found := robin.routes[route.String()]
	robin.lock.RUnlock()
	if found {
		return errors.New("addr already in rotation")
	}

	// Add to our structure
	robin.lock.Lock()
	robin.routes[route.String()] = &orderedRoute{
		route: route,
		index: len(robin.routes),
	}
	robin.lock.Unlock()
	return nil
}

func (robin *RoundRobin) Remove(route Route) error {
	robin.lock.RLock()
	_, found := robin.routes[route.String()]
	robin.lock.RUnlock()
	if !found {
		return errors.New("addr not found")
	}

	defer func() {
		robin.lock.Lock()
		defer robin.lock.Unlock()
		// remove it
		deletedIP := robin.routes[route.String()]
		delete(robin.routes, route.String())

		// update order
		if len(robin.routes) == 0 {
			return
		}

		for _, route := range robin.routes {
			if route.index < deletedIP.index {
				continue
			}
			route.index = route.index - 1
		}
	}()

	return nil
}

func (robin *RoundRobin) Update(routes []Route) {
	robin.lock.Lock()

	robin.routes = make(map[string]*orderedRoute)
	index := 0
	for _, route := range routes {
		if _, found := robin.routes[route.String()]; found {
			continue
		}
		robin.routes[route.String()] = &orderedRoute{
			route: route,
			index: index,
		}
		index++
	}

	robin.lock.Unlock()
}

func (robin *RoundRobin) Get() ([]Route, error) {
	// when done update the order
	// logically (in my mind), I would put this add then end of the func since it should happen last.
	// however since defer is `Last In First Out` we are doing it now before robin.lock.RUnlock().
	defer func() {
		robin.lock.Lock()

		size := len(robin.routes)

		for _, ip := range robin.routes {
			if ip.index == 0 {
				ip.index = size - 1
				continue
			}
			ip.index = ip.index - 1
		}

		robin.lock.Unlock()
	}()

	defer robin.lock.RUnlock()
	robin.lock.RLock()

	records := make([]Route, len(robin.routes))
	if len(robin.routes) == 0 {
		return records, errors.New("no records available")
	}

	for _, route := range robin.routes {
		records[route.index] = route.route
	}

	return records, nil
}

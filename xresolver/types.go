package xresolver

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
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

func CreateRoute(route string) (Route, error) {
	path, err := url.Parse(route)
	if err != nil {
		return Route{}, err
	}
	port, err := strconv.Atoi(path.Port())
	return Route{
		Scheme: path.Scheme,
		Host:   path.Hostname(),
		Port:   port,
	}, err
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

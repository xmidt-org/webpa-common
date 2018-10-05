package xresolver

import (
	"context"
	"errors"
	"net"
	"sync"
)

type Lookup interface {
	// LookupIPAddr looks up host using the local resolver. It returns a slice of that host's IPv4 and IPv6 addresses.
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
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

type orderedIP struct {
	ip    net.IPAddr
	index int
}

type RoundRobin struct {
	lock *sync.RWMutex
	ips  map[string]*orderedIP
}

func NewRoundRobinBalancer() *RoundRobin {
	return &RoundRobin{
		lock: new(sync.RWMutex),
		ips:  make(map[string]*orderedIP, 0),
	}
}

func (robin *RoundRobin) Add(addr net.IPAddr) error {
	// check if exist
	robin.lock.RLock()
	if _, found := robin.ips[addr.String()]; found {
		robin.lock.RUnlock()
		return errors.New("addr already in rotation")
	}
	robin.lock.RUnlock()

	// Add to our structure
	robin.lock.Lock()
	defer robin.lock.Unlock()
	robin.ips[addr.String()] = &orderedIP{
		ip:    addr,
		index: len(robin.ips),
	}
	return nil
}

func (robin *RoundRobin) Remove(addr net.IPAddr) error {
	robin.lock.RLock()
	if _, found := robin.ips[addr.String()]; !found {
		robin.lock.RUnlock()
		return errors.New("addr not found")
	}
	defer func() {
		robin.lock.RUnlock()

		robin.lock.Lock()
		defer robin.lock.Unlock()
		// remove it
		deletedIP := robin.ips[addr.String()]
		delete(robin.ips, addr.String())

		// update order
		if len(robin.ips) == 0 {
			return
		}

		for _, ip := range robin.ips {
			if ip.index < deletedIP.index {
				continue
			}
			ip.index = ip.index - 1
		}
	}()

	return nil

}

func (robin *RoundRobin) Get() ([]net.IPAddr, error) {
	robin.lock.RLock()

	records := make([]net.IPAddr, len(robin.ips))
	if len(robin.ips) == 0 {
		robin.lock.RUnlock()
		return records, errors.New("no records available")
	}

	for _, ip := range robin.ips {
		records[ip.index] = ip.ip
	}

	// update order
	defer func() {
		robin.lock.RUnlock()

		robin.lock.Lock()
		defer robin.lock.Unlock()
		size := len(robin.ips)

		for _, ip := range robin.ips {
			if ip.index == 0 {
				ip.index = size - 1
				continue
			}
			ip.index = ip.index - 1
		}
	}()

	return records, nil
}

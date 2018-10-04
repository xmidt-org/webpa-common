package xresolver

import (
	"context"
	"errors"
	"net"
	"sync"
)

// Note to self: Dial is not being set for net.Resolver because that is the Dial to the DNS server.

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

var DefaultDialer = &net.Dialer{}

type resolver struct {
	resolvers map[Lookup]struct{}
	lock      *sync.RWMutex
	dialer    *net.Dialer
}

func NewResolver(dialer *net.Dialer) Resolver {
	if dialer == nil {
		dialer = DefaultDialer
	}
	return &resolver{
		resolvers: map[Lookup]struct{}{net.DefaultResolver: {}},
		lock:      new(sync.RWMutex),
		dialer:    dialer,
	}
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

func (resolve *resolver) getRecords(ctx context.Context, host string) []net.IPAddr {
	records := make([]net.IPAddr, 0)
	for r := range resolve.resolvers {
		tempRecords, err := r.LookupIPAddr(ctx, host)
		if err == nil {
			records = append(records, tempRecords...)
		}
	}

	return records
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
	records := resolve.getRecords(ctx, host)

	// generate Con or err from records
	con, err = resolve.createConnection(records, network, port)
	if err == nil {
		return
	}

	// if no connection is create use the default dialer
	return resolve.dialer.DialContext(ctx, network, addr)
}

func (resolve *resolver) createConnection(records []net.IPAddr, network, port string) (con net.Conn, err error) {
	for _, item := range records {
		con, err = resolve.dialer.Dial(network, net.JoinHostPort(item.IP.String(), port))
		if err == nil {
			return
		}
	}
	return nil, errors.New("failed to create connection from records")
}

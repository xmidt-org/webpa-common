package service

import (
	"github.com/strava/go.serversets"
	"net"
	"strconv"
)

// RegisteredEndpoints is a mapping of hashed endpoints of the form produced by Accessors
// to strava Endpoint objects.  This data structure is used to carry information about the registration
// of endpoints to client code.
type RegisteredEndpoints map[string]*serversets.Endpoint

// AddHostPort handles producing the same endpoint string as produced by Watches
// and maps that string to the given endpoint object.
func (r RegisteredEndpoints) AddHostPort(host string, port int, endpoint *serversets.Endpoint) {
	hashedEndpoint, _ := ParseHostPort(net.JoinHostPort(host, strconv.Itoa(port)))
	r[hashedEndpoint] = endpoint
}

// Has simply tests if the given watched endpoint occurs in this mapping.
func (r RegisteredEndpoints) Has(hashedEndpoint string) (ok bool) {
	_, ok = r[hashedEndpoint]
	return
}

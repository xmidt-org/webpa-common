package fanout

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/service/monitor"
	"github.com/xmidt-org/webpa-common/service/servicehttp"
	"github.com/xmidt-org/webpa-common/xhttp"
)

// ServiceEndpoints is an Endpoints implementation that is driven by service discovery.
// This type is a monitor.Listener, and fanout URLs are computed from all configured
// instancers.
type ServiceEndpoints struct {
	lock            sync.RWMutex
	keyFunc         servicehttp.KeyFunc
	accessorFactory service.AccessorFactory
	accessors       map[string]service.Accessor
}

// FanoutURLs uses the currently available discovered endpoints to produce a set of URLs.
// The original request is used to produce a hash key, then each accessor is consulted for
// the endpoint that matches that key.
func (se *ServiceEndpoints) FanoutURLs(original *http.Request) ([]*url.URL, error) {
	hashKey, err := se.keyFunc(original)
	if err != nil {
		return nil, err
	}

	se.lock.RLock()
	endpoints := make([]string, 0, len(se.accessors))
	for _, a := range se.accessors {
		e, err := a.Get(hashKey)
		if err != nil {
			continue
		}

		endpoints = append(endpoints, e)
	}

	se.lock.RUnlock()
	if len(endpoints) == 0 {
		return []*url.URL{}, errNoFanoutURLs
	}
	return xhttp.ApplyURLParser(url.Parse, endpoints...)
}

// MonitorEvent supplies the monitor.Listener behavior.  An accessor is created and stored under
// the event Key.
func (se *ServiceEndpoints) MonitorEvent(e monitor.Event) {
	accessor := se.accessorFactory(e.Instances)
	se.lock.Lock()
	se.accessors[e.Key] = accessor
	se.lock.Unlock()
}

// ServiceEndpointsOption is a strategy for configuring a ServiceEndpoints
type ServiceEndpointsOption func(*ServiceEndpoints)

// WithKeyFunc configures a hash function for the given service endpoints.  If nil,
// then device.IDHashParser is used.
func WithKeyFunc(kf servicehttp.KeyFunc) ServiceEndpointsOption {
	return func(se *ServiceEndpoints) {
		if kf != nil {
			se.keyFunc = kf
		} else {
			se.keyFunc = device.IDHashParser
		}
	}
}

// WithAccessorFactory configures an accessor factory for the given service endpoints.
// If nil, then service.DefaultAccessorFactory is used.
func WithAccessorFactory(af service.AccessorFactory) ServiceEndpointsOption {
	return func(se *ServiceEndpoints) {
		if af != nil {
			se.accessorFactory = af
		} else {
			se.accessorFactory = service.DefaultAccessorFactory
		}
	}
}

// NewServiceEndpoints creates a ServiceEndpoints instance.  By default, device.IDHashParser is used as the KeyFunc
// and service.DefaultAccessorFactory is used as the accessor factory.
func NewServiceEndpoints(options ...ServiceEndpointsOption) *ServiceEndpoints {
	se := &ServiceEndpoints{
		keyFunc:         device.IDHashParser,
		accessorFactory: service.DefaultAccessorFactory,
		accessors:       make(map[string]service.Accessor),
	}

	for _, o := range options {
		o(se)
	}

	return se
}

// ServiceEndpointsAlternate creates an alternate closure appropriate for NewEndpoints.  This function
// allows service discovery to be used for fanout endpoints when no fixed endpoints are provided via Options.
//
// The returned Endpoints, being a ServiceEndpoints, will implement monitor.Listener.  This function does not
// start listening to service discovery events.  That must be done by application code.
func ServiceEndpointsAlternate(options ...ServiceEndpointsOption) func() (Endpoints, error) {
	return func() (Endpoints, error) {
		return NewServiceEndpoints(options...), nil
	}
}

// MonitorEndpoints applies service discovery updates to the given Endpoints, if and only if the
// Endpoints implements monitor.Listener.  If Endpoints does not implement monitor.Listener, this
// function return a nil monitor and a nil error.
func MonitorEndpoints(e Endpoints, options ...monitor.Option) (monitor.Interface, error) {
	if l, ok := e.(monitor.Listener); ok {
		options = append(options, monitor.WithListeners(l))
		return monitor.New(options...)
	}

	return nil, nil
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package fanout

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/xmidt-org/webpa-common/v2/device"

	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/service/monitor"
	"github.com/xmidt-org/webpa-common/v2/service/multiaccessor"
	"github.com/xmidt-org/webpa-common/v2/service/servicehttp"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
)

// ServiceEndpoints is an Endpoints implementation that is driven by service discovery.
// This type is a monitor.Listener, and fanout URLs are computed from all configured
// instancers.
type ServiceEndpoints struct {
	lock                 sync.RWMutex
	keyFunc              servicehttp.KeyFunc
	multiAccessorFactory multiaccessor.MultiAccessorFactory
	accessors            map[string]multiaccessor.MultiAccessor
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
		es, err := a.Get(hashKey)
		if err != nil {
			continue
		}

		endpoints = append(endpoints, es...)
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
	accessor := se.multiAccessorFactory(e.Instances)
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

// WithHasherFactory configures an accessor factory for the given service endpoints.
// If nil, then accessor.DefaultAccessorFactory is used.
func WithHasherFactory(hf multiaccessor.MultiAccessorFactory) ServiceEndpointsOption {
	return func(se *ServiceEndpoints) {
		if hf != nil {
			se.multiAccessorFactory = hf
		} else {
			se.multiAccessorFactory = multiaccessor.DefaultMultiAccessorFactory
		}
	}
}

// NewServiceEndpoints creates a ServiceEndpoints instance.  By default, device.IDHashParser is used as the KeyFunc
// and accessor.DefaultAccessorFactory is used as the accessor factory.
func NewServiceEndpoints(options ...ServiceEndpointsOption) *ServiceEndpoints {
	se := &ServiceEndpoints{
		keyFunc:              device.IDHashParser,
		multiAccessorFactory: multiaccessor.DefaultMultiAccessorFactory,
		accessors:            make(map[string]multiaccessor.MultiAccessor),
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

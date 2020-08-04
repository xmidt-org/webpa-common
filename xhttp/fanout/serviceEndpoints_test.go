package fanout

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/service/monitor"
)

func testNewServiceEndpointsHashError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		request = httptest.NewRequest("GET", "/", nil)

		se = NewServiceEndpoints()
	)

	require.NotNil(se)
	request.Header.Set(device.DeviceNameHeader, "mac:112233445566")

	urls, err := se.FanoutURLs(request)
	assert.Empty(urls)
	assert.Error(err)
}

func testNewServiceEndpointsKeyFuncError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		request = httptest.NewRequest("GET", "/", nil)

		expectedError = errors.New("expected error from KeyFunc")
		keyFunc       = func(r *http.Request) ([]byte, error) {
			return nil, expectedError
		}

		se = NewServiceEndpoints(WithKeyFunc(keyFunc))
	)

	require.NotNil(se)
	urls, err := se.FanoutURLs(request)
	assert.Empty(urls)
	assert.Equal(expectedError, err)
}

func testNewServiceEndpointsDefault(t *testing.T, se *ServiceEndpoints) {
	var (
		assert  = assert.New(t)
		request = httptest.NewRequest("GET", "/", nil)
	)

	request.Header.Set(device.DeviceNameHeader, "mac:112233445566")

	urls, err := se.FanoutURLs(request)
	assert.Empty(urls)
	assert.Error(err)

	se.MonitorEvent(monitor.Event{Key: "key1"})
	urls, err = se.FanoutURLs(request)
	assert.Empty(urls)
	assert.Error(err)

	se.MonitorEvent(monitor.Event{Key: "key1", Instances: []string{"http://localhost:8080"}})
	urls, err = se.FanoutURLs(request)
	assert.Len(urls, 1)
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "localhost:8080"})
	assert.NoError(err)

	se.MonitorEvent(monitor.Event{Key: "key2", Instances: []string{"http://foobar.net:1234"}})
	urls, err = se.FanoutURLs(request)
	assert.Len(urls, 2)
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "localhost:8080"})
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "foobar.net:1234"})
	assert.NoError(err)

	se.MonitorEvent(monitor.Event{Key: "key1", Instances: []string{"https://somewhere.com"}})
	urls, err = se.FanoutURLs(request)
	assert.Len(urls, 2)
	assert.Contains(urls, &url.URL{Scheme: "https", Host: "somewhere.com"})
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "foobar.net:1234"})
	assert.NoError(err)
}

func testNewServiceEndpointsCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		request = httptest.NewRequest("GET", "/", nil)

		keyFuncCalled = false
		keyFunc       = func(r *http.Request) ([]byte, error) {
			keyFuncCalled = true
			return device.IDHashParser(r)
		}

		accessorFactoryCalled = false
		accessorFactory       = func(instances []string) service.Accessor {
			accessorFactoryCalled = true
			return service.DefaultAccessorFactory(instances)
		}

		se = NewServiceEndpoints(WithAccessorFactory(accessorFactory), WithKeyFunc(keyFunc))
	)

	require.NotNil(se)
	request.Header.Set(device.DeviceNameHeader, "mac:112233445566")

	urls, err := se.FanoutURLs(request)
	assert.True(keyFuncCalled)
	assert.Empty(urls)
	assert.Error(err)

	keyFuncCalled = false
	accessorFactoryCalled = false
	se.MonitorEvent(monitor.Event{Key: "key1"})
	assert.True(accessorFactoryCalled)
	urls, err = se.FanoutURLs(request)
	assert.Empty(urls)
	assert.Error(err)

	keyFuncCalled = false
	accessorFactoryCalled = false
	se.MonitorEvent(monitor.Event{Key: "key1", Instances: []string{"http://localhost:8080"}})
	urls, err = se.FanoutURLs(request)
	assert.True(keyFuncCalled)
	assert.True(accessorFactoryCalled)
	assert.Len(urls, 1)
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "localhost:8080"})
	assert.NoError(err)

	keyFuncCalled = false
	accessorFactoryCalled = false
	se.MonitorEvent(monitor.Event{Key: "key2", Instances: []string{"http://foobar.net:1234"}})
	urls, err = se.FanoutURLs(request)
	assert.True(keyFuncCalled)
	assert.True(accessorFactoryCalled)
	assert.Len(urls, 2)
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "localhost:8080"})
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "foobar.net:1234"})
	assert.NoError(err)

	keyFuncCalled = false
	accessorFactoryCalled = false
	se.MonitorEvent(monitor.Event{Key: "key1", Instances: []string{"https://somewhere.com"}})
	urls, err = se.FanoutURLs(request)
	assert.True(keyFuncCalled)
	assert.True(accessorFactoryCalled)
	assert.Len(urls, 2)
	assert.Contains(urls, &url.URL{Scheme: "https", Host: "somewhere.com"})
	assert.Contains(urls, &url.URL{Scheme: "http", Host: "foobar.net:1234"})
	assert.NoError(err)
}

func TestNewServiceEndpoints(t *testing.T) {
	t.Run("KeyFuncError", testNewServiceEndpointsKeyFuncError)

	t.Run("Default", func(t *testing.T) {
		testNewServiceEndpointsDefault(t, NewServiceEndpoints())
		testNewServiceEndpointsDefault(t, NewServiceEndpoints(WithAccessorFactory(nil), WithKeyFunc(nil)))
	})

	t.Run("Custom", testNewServiceEndpointsCustom)
}

func TestServiceEndpointsAlternate(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		e, err = ServiceEndpointsAlternate()()
	)

	require.NotNil(e)
	assert.NoError(err)

	se, ok := e.(*ServiceEndpoints)
	require.True(ok)

	assert.NotNil(se.keyFunc)
	assert.NotNil(se.accessorFactory)
}

func testMonitorListenerWithNonListener(t *testing.T) {
	var (
		assert = assert.New(t)

		fe     = FixedEndpoints{}
		m, err = MonitorEndpoints(fe)
	)

	assert.Nil(m)
	assert.NoError(err)
}

func testMonitorListenerWithListener(t *testing.T) {
	var (
		assert = assert.New(t)

		deregisterWait = make(chan struct{})

		i  = new(service.MockInstancer)
		se = NewServiceEndpoints()
	)

	i.On("Register", mock.MatchedBy(func(chan<- sd.Event) bool { return true })).Once()
	i.On("Deregister", mock.MatchedBy(func(chan<- sd.Event) bool { return true })).Once().Run(func(mock.Arguments) {
		close(deregisterWait)
	})

	m, err := MonitorEndpoints(se, monitor.WithInstancers(service.Instancers{"key": i}))
	assert.NotNil(m)
	assert.NoError(err)
	m.Stop()

	<-deregisterWait
	i.AssertExpectations(t)
}

func TestMonitorEndpoints(t *testing.T) {
	t.Run("WithNonListener", testMonitorListenerWithNonListener)
	t.Run("WithListener", testMonitorListenerWithListener)
}

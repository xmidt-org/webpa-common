package consul

import (
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/adapter"
)

var log = adapter.Logger{
	Logger: sallust.Default(),
}

func TestDefaultTickerFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	assert.Panics(func() {
		defaultTickerFactory(-123123)
	})

	ticker, stop := defaultTickerFactory(20 * time.Second)
	assert.NotNil(ticker)
	require.NotNil(stop)
	stop()
}

func testNewRegistrarNoChecks(t *testing.T) {
	defer resetTickerFactory()

	var (
		require       = require.New(t)
		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
		}
	)

	client.On("Register",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	client.On("Deregister",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	require.NoError(err)
	require.NotNil(r)

	r.Register()
	r.Deregister()

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarNoTTL(t *testing.T) {
	defer resetTickerFactory()

	var (
		require = require.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Check: &api.AgentServiceCheck{
				CheckID: "check1",
				HTTP:    "https://foobar.com/foo",
			},
			Checks: []*api.AgentServiceCheck{
				{
					CheckID: "check2",
					HTTP:    "https://foobar.com/moo",
				},
			},
		}
	)

	client.On("Register",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	client.On("Deregister",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	require.NoError(err)
	require.NotNil(r)

	r.Register()
	r.Deregister()

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarCheckMalformedTTL(t *testing.T) {
	defer resetTickerFactory()

	var (
		assert = assert.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Check: &api.AgentServiceCheck{
				CheckID: "check1",
				TTL:     "this is not valid",
			},
		}
	)

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	assert.Error(err)
	assert.Nil(r)

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarCheckTTLTooSmall(t *testing.T) {
	defer resetTickerFactory()

	var (
		assert = assert.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Check: &api.AgentServiceCheck{
				CheckID: "check1",
				TTL:     "1ns",
			},
		}
	)

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	assert.Error(err)
	assert.Nil(r)

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarChecksMalformedTTL(t *testing.T) {
	defer resetTickerFactory()

	var (
		assert = assert.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Checks: []*api.AgentServiceCheck{
				{
					CheckID: "check1",
					TTL:     "this is not valid",
				},
			},
		}
	)

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	assert.Error(err)
	assert.Nil(r)

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarChecksTTLTooSmall(t *testing.T) {
	defer resetTickerFactory()

	var (
		assert = assert.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Checks: []*api.AgentServiceCheck{
				{
					CheckID: "check1",
					TTL:     "1ns",
				},
			},
		}
	)

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	assert.Error(err)
	assert.Nil(r)

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func testNewRegistrarTTL(t *testing.T) {
	defer resetTickerFactory()

	var (
		assert  = assert.New(t)
		require = require.New(t)

		client        = new(mockClient)
		ttlUpdater    = new(mockTTLUpdater)
		tickerFactory = prepareMockTickerFactory()

		timer1       = make(chan time.Time, 1)
		timer1Ack    = make(chan struct{}, 1)
		timer1AckRun = func(mock.Arguments) { timer1Ack <- struct{}{} }
		update1Done  = make(chan struct{})
		stop1        = func() {
			close(update1Done)
		}

		timer2       = make(chan time.Time, 1)
		timer2Ack    = make(chan struct{}, 1)
		timer2AckRun = func(mock.Arguments) { timer2Ack <- struct{}{} }
		update2Done  = make(chan struct{})
		stop2        = func() {
			close(update2Done)
		}

		registration = &api.AgentServiceRegistration{
			ID:      "service1",
			Address: "somehost.com",
			Port:    1111,
			Check: &api.AgentServiceCheck{
				CheckID: "check1",
				TTL:     "15s",
			},
			Checks: []*api.AgentServiceCheck{
				{
					CheckID: "check2",
					TTL:     "30s",
				},
			},
		}
	)

	ttlUpdater.On("UpdateTTL", "check1", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(error(nil)).Once().Run(timer1AckRun)
	ttlUpdater.On("UpdateTTL", "check1", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(errors.New("expected check1 error")).Once().Run(timer1AckRun)
	ttlUpdater.On("UpdateTTL", "check1", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(error(nil)).Once().Run(timer1AckRun)
	ttlUpdater.On("UpdateTTL", "check1", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "fail").Return(error(nil)).Once()

	ttlUpdater.On("UpdateTTL", "check2", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(error(nil)).Once().Run(timer2AckRun)
	ttlUpdater.On("UpdateTTL", "check2", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(errors.New("expected check2 error")).Once().Run(timer2AckRun)
	ttlUpdater.On("UpdateTTL", "check2", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "pass").Return(error(nil)).Once().Run(timer2AckRun)
	ttlUpdater.On("UpdateTTL", "check2", mock.MatchedBy(func(v string) bool { return len(v) > 0 }), "fail").Return(errors.New("expected check2 fail error")).Once()

	tickerFactory.On("NewTicker", (15*time.Second)/2).Return((<-chan time.Time)(timer1), stop1)
	tickerFactory.On("NewTicker", (30*time.Second)/2).Return((<-chan time.Time)(timer2), stop2)

	client.On("Register",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	client.On("Deregister",
		mock.MatchedBy(func(r *api.AgentServiceRegistration) bool {
			return r.ID == "service1"
		}),
	).Return(error(nil)).Once()

	r, err := NewRegistrar(client, ttlUpdater, registration, log)
	require.NoError(err)
	require.NotNil(r)

	r.Register()
	r.Register() // idempotent

	// simulate some updates
	now := time.Now()

	// we have 3 pass updates expected for each TTL check above
	for repeat := 0; repeat < 3; repeat++ {
		timer1 <- now
		select {
		case <-timer1Ack:
			// passing
		case <-time.After(2 * time.Second):
			require.Fail("Time event was not processed")
		}

		timer2 <- now
		select {
		case <-timer2Ack:
			// passing
		case <-time.After(2 * time.Second):
			require.Fail("Time event was not processed")
		}
	}

	r.Deregister()
	r.Deregister() // idempotent

	select {
	case <-update1Done:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("TTL update goroutine did not fail the TTL")
	}

	select {
	case <-update2Done:
		// passing
	case <-time.After(2 * time.Second):
		assert.Fail("TTL update goroutine did not fail the TTL")
	}

	client.AssertExpectations(t)
	ttlUpdater.AssertExpectations(t)
	tickerFactory.AssertExpectations(t)
}

func TestNewRegistrar(t *testing.T) {
	t.Run("NoChecks", testNewRegistrarNoChecks)
	t.Run("NoTTL", testNewRegistrarNoTTL)

	t.Run("Check", func(t *testing.T) {
		t.Run("MalformedTTL", testNewRegistrarCheckMalformedTTL)
		t.Run("TTLTooSmall", testNewRegistrarCheckTTLTooSmall)
	})

	t.Run("Checks", func(t *testing.T) {
		t.Run("MalformedTTL", testNewRegistrarChecksMalformedTTL)
		t.Run("TTLTooSmall", testNewRegistrarChecksTTLTooSmall)
	})

	t.Run("TTL", testNewRegistrarTTL)
}

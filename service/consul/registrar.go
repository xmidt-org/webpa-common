package consul

import (
	"fmt"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

// passFormat returns a closure that produces the output for a passing TTL, given the current system time
func passFormat(serviceID string) func(time.Time) string {
	return func(t time.Time) string {
		return fmt.Sprintf("%s passed at %s", serviceID, t.UTC())
	}
}

// failFormat returns a closure that produces the output for a critical TTL, given the current system time
func failFormat(serviceID string) func(time.Time) string {
	return func(t time.Time) string {
		return fmt.Sprintf("%s failed at %s", serviceID, t.UTC())
	}
}

func defaultTickerFactory(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

var tickerFactory = defaultTickerFactory

// ttlUpdater represents any object which can update the TTL status on the remote consul cluster.
// The consul api Client implements this interface.
type ttlUpdater interface {
	UpdateTTL(checkID, output, status string) error
}

// ttlCheck holds the relevant information for managing a TTL check
type ttlCheck struct {
	checkID    string
	interval   time.Duration
	logger     log.Logger
	passFormat func(time.Time) string
	failFormat func(time.Time) string
}

func (tc ttlCheck) updatePeriodically(updater ttlUpdater, shutdown <-chan struct{}) {
	ticker, stop := tickerFactory(tc.interval)
	defer stop()
	defer func() {
		if err := updater.UpdateTTL(tc.checkID, tc.failFormat(time.Now()), "fail"); err != nil {
			tc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error while updating TTL to critical", logging.ErrorKey(), err)
		}
	}()

	tc.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "starting TTL updater")

	// we log an error only on the first error, and then an info message if and when the update recovers.
	// this avoids filling up the server's logs with what are almost certainly just duplicate errors over and over.
	successiveErrorCount := 0

	for {
		select {
		case t := <-ticker:
			if err := updater.UpdateTTL(tc.checkID, tc.passFormat(t), "pass"); err != nil {
				successiveErrorCount++
				if successiveErrorCount == 1 {
					tc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error while updating TTL to passing", logging.ErrorKey(), err)
				}
			} else if successiveErrorCount > 0 {
				tc.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "update TTL success", "previousErrorCount", successiveErrorCount)
				successiveErrorCount = 0
			}

		case <-shutdown:
			tc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "TTL updater shutdown")
			return
		}
	}
}

// appendTTLCheck conditionally creates a ttlCheck for the given agent check if and only if the agent check is configured with a TTL.
// If the agent check is nil or has no TTL, this function returns ttlChecks unmodified with no error.
func appendTTLCheck(logger log.Logger, serviceID string, agentCheck *api.AgentServiceCheck, ttlChecks []ttlCheck) ([]ttlCheck, error) {
	if agentCheck == nil || len(agentCheck.TTL) == 0 {
		return ttlChecks, nil
	}

	ttl, err := time.ParseDuration(agentCheck.TTL)
	if err != nil {
		return nil, err
	}

	interval := ttl / 2
	if interval < 1 {
		return nil, fmt.Errorf("TTL %s is too small", agentCheck.TTL)
	}

	ttlChecks = append(
		ttlChecks,
		ttlCheck{
			checkID:  agentCheck.CheckID,
			interval: interval,
			logger: log.With(
				logger,
				"serviceID", serviceID,
				"checkID", agentCheck.CheckID,
				"ttl", agentCheck.TTL,
				"interval", interval.String(),
			),
			passFormat: passFormat(serviceID),
			failFormat: failFormat(serviceID),
		},
	)

	return ttlChecks, nil
}

// ttlRegistrar is an sd.Registrar that binds one or more TTL updates to the Register/Deregister lifecycle.
// When Register is called, a goroutine is spawned for each TTL check that invokes UpdateTTL on an interval.
// When Dereigster is called, any goroutines spawned are stopped and each check is set to fail (critical).
type ttlRegistrar struct {
	logger    log.Logger
	serviceID string
	registrar sd.Registrar
	updater   ttlUpdater
	checks    []ttlCheck

	lifecycleLock sync.Mutex
	shutdown      chan struct{}
}

// NewRegistrar creates an sd.Registrar, binding any TTL checks to the Register/Deregister lifecycle as needed.
func NewRegistrar(c gokitconsul.Client, u ttlUpdater, r *api.AgentServiceRegistration, logger log.Logger) (sd.Registrar, error) {
	var (
		ttlChecks []ttlCheck
		err       error
	)

	ttlChecks, err = appendTTLCheck(logger, r.ID, r.Check, ttlChecks)
	if err != nil {
		return nil, err
	}

	for _, agentCheck := range r.Checks {
		ttlChecks, err = appendTTLCheck(logger, r.ID, agentCheck, ttlChecks)
		if err != nil {
			return nil, err
		}
	}

	var registrar sd.Registrar = gokitconsul.NewRegistrar(c, r, logger)

	// decorate the given registrar if we have any TTL checks
	if len(ttlChecks) > 0 {
		registrar = &ttlRegistrar{
			logger:    logger,
			serviceID: r.ID,
			registrar: registrar,
			updater:   u,
			checks:    ttlChecks,
		}
	}

	return registrar, nil
}

func (tr *ttlRegistrar) Register() {
	defer tr.lifecycleLock.Unlock()
	tr.lifecycleLock.Lock()

	if tr.shutdown != nil {
		return
	}

	tr.registrar.Register()
	tr.shutdown = make(chan struct{})
	for _, tc := range tr.checks {
		go tc.updatePeriodically(tr.updater, tr.shutdown)
	}
}

func (tr *ttlRegistrar) Deregister() {
	defer tr.lifecycleLock.Unlock()
	tr.lifecycleLock.Lock()

	if tr.shutdown == nil {
		return
	}

	close(tr.shutdown)
	tr.shutdown = nil
	tr.registrar.Deregister()
}

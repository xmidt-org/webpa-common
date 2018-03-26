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

type tickerFactory func(time.Duration) (<-chan time.Time, func())

func defaultTickerFactory(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

// ttlUpdater represents any object which can update the TTL status on the remote consul cluster.
// The consul api Client implements this interface.
type ttlUpdater interface {
	UpdateTTL(checkID, output, status string) error
}

// ttlCheck holds the relevant information for managing a TTL check
type ttlCheck struct {
	checkID  string
	interval time.Duration
}

// appendTTLCheck conditionally creates a ttlCheck for the given agent check if and only if the agent check is configured with a TTL.
// If the agent check is nil or has no TTL, this function returns ttlChecks unmodified with no error.
func appendTTLCheck(agentCheck *api.AgentServiceCheck, ttlChecks []ttlCheck) ([]ttlCheck, error) {
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
	tf        tickerFactory

	lifecycleLock sync.Mutex
	shutdown      chan struct{}
}

// NewRegistrar creates an sd.Registrar, binding any TTL checks to the Register/Deregister lifecycle as needed.
func NewRegistrar(c gokitconsul.Client, u ttlUpdater, r *api.AgentServiceRegistration, logger log.Logger) (sd.Registrar, error) {
	var (
		ttlChecks []ttlCheck
		err       error
	)

	ttlChecks, err = appendTTLCheck(r.Check, ttlChecks)
	if err != nil {
		return nil, err
	}

	for _, agentCheck := range r.Checks {
		ttlChecks, err = appendTTLCheck(agentCheck, ttlChecks)
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
			tf:        defaultTickerFactory,
		}
	}

	return registrar, nil
}

func (tr *ttlRegistrar) updatePeriodically(tc ttlCheck, shutdown <-chan struct{}) {
	ticker, stop := tr.tf(tc.interval)
	defer stop()
	defer func() {
		tr.updater.UpdateTTL(tc.checkID, fmt.Sprintf("%s failed at %s", tr.serviceID, time.Now().UTC()), "fail")
	}()

	for {
		select {
		case t := <-ticker:
			if err := tr.updater.UpdateTTL(tc.checkID, fmt.Sprintf("%s passed at %s", tr.serviceID, t.UTC()), "pass"); err != nil {
				tr.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "failed to update TTL", logging.ErrorKey(), err)
			}

		case <-shutdown:
			tr.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "TTL updater shutdown")
			return
		}
	}
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
		go tr.updatePeriodically(tc, tr.shutdown)
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

package health

import (
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/c9s/goprocinfo/linux"
	"net/http"
	"runtime"
	"sync"
	"time"
)

const (
	CurrentMemoryUtilizationAlloc   Stat = "CurrentMemoryUtilizationAlloc"
	CurrentMemoryUtilizationHeapSys Stat = "CurrentMemoryUtilizationHeapSys"
	CurrentMemoryUtilizationActive  Stat = "CurrentMemoryUtilizationActive"
	MaxMemoryUtilizationAlloc       Stat = "MaxMemoryUtilizationAlloc"
	MaxMemoryUtilizationHeapSys     Stat = "MaxMemoryUtilizationHeapSys"
	MaxMemoryUtilizationActive      Stat = "MaxMemoryUtilizationActive"
)

// commonStats is the Stats used to seed the initial set of stats
var commonStats = Stats{
	CurrentMemoryUtilizationAlloc:   0,
	CurrentMemoryUtilizationHeapSys: 0,
	CurrentMemoryUtilizationActive:  0,
	MaxMemoryUtilizationAlloc:       0,
	MaxMemoryUtilizationHeapSys:     0,
	MaxMemoryUtilizationActive:      0,
}

// production code:
type OsChecker interface {
	OsName() string
}

type defaultOsChecker struct {
}

func (d defaultOsChecker) OsName() string {
	return runtime.GOOS
}

// StatsListener receives Stats on regular intervals.
type StatsListener interface {
	// OnStats is called with a copy of the health's stats map
	// at regular intervals.
	OnStats(Stats)
}

// StatsListenerFunc is a function type that implements StatsListener.
type StatsListenerFunc func(Stats)

func (f StatsListenerFunc) OnStats(stats Stats) {
	f(stats)
}

// HealthFunc functions are allowed to modify the passed-in stats.
type HealthFunc func(Stats)

// Stat is a named piece of data to be tracked
type Stat string

// Stats is mapping of Stat to value
type Stats map[Stat]int

// Health is the central type of this package.  It defines and endpoint for tracking
// and updating various statistics.  It also dispatches events to one or more StatsListeners
// at regular intervals.
type Health struct {
	stats            Stats
	statDumpInterval time.Duration
	log              logging.Logger
	wg               *sync.WaitGroup
	event            chan HealthFunc
	osChecker        OsChecker
	statsListeners   []StatsListener
}

// AddStatsListener adds a new listener to this Health.  This method
// is asynchronous.  The listener will eventually receive events, but callers
// should not assume events will be dispatched immediately after this method call.
func (h *Health) AddStatsListener(listener StatsListener) {
	h.SendEvent(func(stat Stats) {
		h.statsListeners = append(h.statsListeners, listener)
	})
}

// SendEvent dispatches a HealthFunc to the internal event queue
func (h *Health) SendEvent(healthFunc HealthFunc) {
	h.event <- healthFunc
}

// Bundle produces an aggregate HealthFunc from a number of others
func Bundle(hfs ...HealthFunc) HealthFunc {
	return func(stats Stats) {
		for _, hf := range hfs {
			hf(stats)
		}
	}
}

// Ensure makes certain the given stat is defined.  If it does not exist,
// it is initialized to 0.  Otherwise, the existing stat value is left intact.
func Ensure(stat Stat) HealthFunc {
	return func(stats Stats) {
		if _, ok := stats[stat]; !ok {
			stats[stat] = 0
		}
	}
}

// Inc increments the given stat by a certain amount
func Inc(stat Stat, value int) HealthFunc {
	return func(stats Stats) {
		stats[stat] += value
	}
}

// Set changes (or, initializes) the stat to the given value
func Set(stat Stat, value int) HealthFunc {
	return func(stats Stats) {
		stats[stat] = value
	}
}

// Close shuts down the health event monitoring
func (h *Health) Close() error {
	close(h.event)
	return nil
}

// New creates a Health object with the given statistics.  This function starts the internal
// monitor goroutine, which will invoke Add(1) on startup and Done() when the returned Health
// is closed.
func New(interval time.Duration, log logging.Logger, wg *sync.WaitGroup, options ...HealthFunc) *Health {
	initialStats := make(Stats, len(commonStats)+len(options))
	for stat, value := range commonStats {
		initialStats[stat] = value
	}

	for _, option := range options {
		option(initialStats)
	}

	h := &Health{
		event:            make(chan HealthFunc, 100),
		stats:            initialStats,
		statDumpInterval: interval,
		log:              log,
		wg:               wg,
		osChecker:        &defaultOsChecker{},
	}

	h.monitor()
	return h
}

func (h *Health) oscheck() bool {
	if h.osChecker.OsName() == "linux" {
		h.log.Debug("Linux operating system detected: %v", runtime.GOOS)
		return true
	} else {
		h.log.Debug("Other operating system detected: %v", runtime.GOOS)
		return false
	}
}

func (h *Health) memory() {
	if h.oscheck() {
		meminfo, err := linux.ReadMemInfo("/proc/meminfo")
		if err != nil {
			h.log.Error("error querying memory information: %v", err)
		} else {
			active := int(meminfo.Active * 1024)
			h.stats[CurrentMemoryUtilizationActive] = active
			if active > h.stats[MaxMemoryUtilizationActive] {
				h.stats[MaxMemoryUtilizationActive] = active
			}
		}

		var memstats runtime.MemStats
		runtime.ReadMemStats(&memstats)
		alloc := int(memstats.Alloc)
		heapsys := int(memstats.HeapSys)

		// set current
		h.stats[CurrentMemoryUtilizationAlloc] = alloc
		h.stats[CurrentMemoryUtilizationHeapSys] = heapsys

		// set max
		if alloc > h.stats[MaxMemoryUtilizationAlloc] {
			h.stats[MaxMemoryUtilizationAlloc] = alloc
		}
		if heapsys > h.stats[MaxMemoryUtilizationHeapSys] {
			h.stats[MaxMemoryUtilizationHeapSys] = heapsys
		}
	}
}

func (h *Health) monitor() {
	h.log.Debug("Health Monitor Started")

	h.wg.Add(1)
	go func() {
		ticker := time.NewTicker(h.statDumpInterval)

		defer ticker.Stop()
		defer h.log.Debug("Health Monitor Stopped")
		defer h.wg.Done()

		for {
			select {
			case hf, ok := <-h.event:
				if !ok {
					return
				}

				hf(h.stats)
			case <-ticker.C:
				hs := h.getStats()
				for _, statsListener := range h.statsListeners {
					statsListener.OnStats(hs)
				}
			}
		}
	}()
}

func (h *Health) getStats() Stats {
	statsCopy := make(Stats)

	h.memory()
	for k, v := range h.stats {
		statsCopy[k] = v
	}

	return statsCopy
}

func (h *Health) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ch := make(chan Stats)
	defer close(ch)

	h.SendEvent(func(stat Stats) {
		ch <- h.getStats()
	})

	hs := <-ch
	jsonmsg, err := json.Marshal(hs)

	if err != nil {
		responseErrorJson(rw, err.Error(), http.StatusInternalServerError, h.log)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(jsonmsg)
}

func responseErrorJson(rw http.ResponseWriter, errmsg string, code int, log logging.Logger) {
	log.Error("Response error code %v msg [%v]", code, errmsg)
	rw.Header().Set("Content-Type", "application/json")
	jsonStr := fmt.Sprintf(`{"message":"%s"}`, errmsg)

	rw.WriteHeader(code)
	fmt.Fprintln(rw, jsonStr)

	return
}

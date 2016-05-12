package health

import (
	"encoding/json"
	"errors"
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

var (
	ErrorCannotReadMemory = errors.New("Cannot read memory")

	// commonStats is the Stats used to seed the initial set of stats
	commonStats = Stats{
		CurrentMemoryUtilizationAlloc:   0,
		CurrentMemoryUtilizationHeapSys: 0,
		CurrentMemoryUtilizationActive:  0,
		MaxMemoryUtilizationAlloc:       0,
		MaxMemoryUtilizationHeapSys:     0,
		MaxMemoryUtilizationActive:      0,
	}
)

// OsChecker returns the name of the underlying operating system.
type OsChecker interface {
	OsName() string
}

// defaultOsChecker is the default implementation of OsChecker.
// This implementation simply delegates to runtime.GOOS.
type defaultOsChecker struct {
}

func (d defaultOsChecker) OsName() string {
	return runtime.GOOS
}

// DefaultOsChecker returns a default implementation of OsChecker,
// which delegates to runtime.GOOS.
func DefaultOsChecker() OsChecker {
	return defaultOsChecker{}
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

// Clone returns a distinct copy of this Stats object
func (s Stats) Clone() Stats {
	copyOf := make(Stats, len(s))
	for key, value := range s {
		copyOf[key] = value
	}

	return copyOf
}

// Apply invokes each HealthFunc on this stats
func (s Stats) Apply(options ...HealthFunc) {
	for _, option := range options {
		option(s)
	}
}

// Health is the central type of this package.  It defines and endpoint for tracking
// and updating various statistics.  It also dispatches events to one or more StatsListeners
// at regular intervals.
type Health struct {
	stats            Stats
	statDumpInterval time.Duration
	log              logging.Logger
	event            chan HealthFunc
	statsListeners   []StatsListener
	memory           HealthFunc
	once             sync.Once
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

// Memory returns a HealthFunc that updates the given stats with memory statistics,
// based on the operation system name.  If the memory was not able to be read, the
// returned HealthFunc will panic.
func Memory(log logging.Logger, osChecker OsChecker) HealthFunc {
	osName := osChecker.OsName()
	log.Info("Operating system detected: %s", osName)

	switch osName {
	case "linux":
		return func(stats Stats) {
			meminfo, err := linux.ReadMemInfo("/proc/meminfo")
			if err != nil {
				log.Error("error querying memory information: %v", err)
				panic(ErrorCannotReadMemory)
			}

			active := int(meminfo.Active * 1024)
			stats[CurrentMemoryUtilizationActive] = active
			if active > stats[MaxMemoryUtilizationActive] {
				stats[MaxMemoryUtilizationActive] = active
			}

			var memstats runtime.MemStats
			runtime.ReadMemStats(&memstats)
			alloc := int(memstats.Alloc)
			heapsys := int(memstats.HeapSys)

			// set current
			stats[CurrentMemoryUtilizationAlloc] = alloc
			stats[CurrentMemoryUtilizationHeapSys] = heapsys

			// set max
			if alloc > stats[MaxMemoryUtilizationAlloc] {
				stats[MaxMemoryUtilizationAlloc] = alloc
			}

			if heapsys > stats[MaxMemoryUtilizationHeapSys] {
				stats[MaxMemoryUtilizationHeapSys] = heapsys
			}
		}
	default:
		// return a noop
		return func(Stats) {}
	}
}

// Close shuts down the health event monitoring
func (h *Health) Close() error {
	close(h.event)
	return nil
}

// New creates a Health object with the given statistics.
func New(interval time.Duration, log logging.Logger, options ...HealthFunc) *Health {
	initialStats := commonStats.Clone()
	initialStats.Apply(options...)

	return &Health{
		event:            make(chan HealthFunc, 100),
		stats:            initialStats,
		statDumpInterval: interval,
		log:              log,
		memory:           Memory(log, DefaultOsChecker()),
	}
}

// Run executes this Health object.  This method is idempotent:  once a
// Health object is Run, it cannot be Run again.
func (h *Health) Run(waitGroup *sync.WaitGroup) {
	h.once.Do(func() {
		h.log.Debug("Health Monitor Started")

		waitGroup.Add(1)
		go func() {
			ticker := time.NewTicker(h.statDumpInterval)

			defer ticker.Stop()
			defer h.log.Debug("Health Monitor Stopped")
			defer waitGroup.Done()

			for {
				select {
				case hf, ok := <-h.event:
					if !ok {
						return
					}

					hf(h.stats)
				case <-ticker.C:
					h.memory(h.stats)
					hs := h.stats.Clone()
					for _, statsListener := range h.statsListeners {
						statsListener.OnStats(hs)
					}
				}
			}
		}()
	})
}

func (h *Health) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	ch := make(chan Stats)
	defer close(ch)

	h.SendEvent(func(stats Stats) {
		h.memory(stats)
		jsonmsg, err := json.Marshal(stats)
		response.Header().Set("Content-Type", "application/json")

		// TODO: leverage the standard error writing elsewhere in webpa-common
		if err != nil {
			h.log.Error("Could not marshal stats: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(response, `{"message": "%s"}\n`, err.Error())
		} else {
			fmt.Fprintf(response, "%s", jsonmsg)
		}
	})
}

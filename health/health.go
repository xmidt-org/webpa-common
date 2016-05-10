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

const CurrentMemoryUtilizationAlloc Stat = "CurrentMemoryUtilizationAlloc"
const CurrentMemoryUtilizationHeapSys Stat = "CurrentMemoryUtilizationHeapSys"
const CurrentMemoryUtilizationActive Stat = "CurrentMemoryUtilizationActive"
const MaxMemoryUtilizationAlloc Stat = "MaxMemoryUtilizationAlloc"
const MaxMemoryUtilizationHeapSys Stat = "MaxMemoryUtilizationHeapSys"
const MaxMemoryUtilizationActive Stat = "MaxMemoryUtilizationActive"

// production code:
type OsChecker interface {
	OsName() string
}

type defaultOsChecker struct {
}

func (d defaultOsChecker) OsName() string {
	return runtime.GOOS
}

// *_test.go code:
type testOsChecker struct {
	osName string
}

func (t testOsChecker) OsName() string {
	return t.osName
}

type StatsListener interface {
	OnStats(Stats)
}

// use for monitoring the stat data
// pass in a StatListenerFunc to SendEvent
type StatsListenerFunc func(Stats)

// use to modify the stat data
// pass in a HealthFunc to SendEvent
type HealthFunc func(Stats)

// a named piece of data to be tracked
type Stat string

// a map of named Stats with corresponding values
type Stats map[Stat]int

type Health struct {
	stats            Stats
	statDumpInterval time.Duration
	log              logging.Logger
	wg               *sync.WaitGroup
	event            chan HealthFunc
	osChecker        OsChecker
	statsListeners   []StatsListener
}

func (h *Health) AddStatsListener(listener StatsListener) {
	h.SendEvent(func(stat Stats) {
		h.statsListeners = append(h.statsListeners, listener)
	})
}

func (f StatsListenerFunc) OnStats(stats Stats) {
	f(stats)
}

// Send types with func(Stats) signatures through here to execute and prevent race conditions
func (h *Health) SendEvent(fn func(Stats)) {
	h.event <- fn
}

func Bundle(hfs ...HealthFunc) HealthFunc {
	return func(stats Stats) {
		for _, hf := range hfs {
			hf(stats)
		}
	}
}

func Inc(stat Stat, value int) HealthFunc {
	return func(stats Stats) {
		stats[stat] += value
	}
}

func Set(stat Stat, value int) HealthFunc {
	return func(stats Stats) {
		stats[stat] = value
	}
}

func (h *Health) Close() {
	close(h.event)
}

func New(interval time.Duration, log logging.Logger, wg *sync.WaitGroup) *Health {
	h := new(Health)
	h.event = make(chan HealthFunc, 100)
	h.stats = make(Stats)
	h.statDumpInterval = interval
	h.log = log
	h.wg = wg
	h.osChecker = new(defaultOsChecker)
	h.monitor()
	h.commonStats()

	return h
}

func (h *Health) commonStats() {
	h.SendEvent(
		Bundle(
			Set(CurrentMemoryUtilizationAlloc, 0),
			Set(CurrentMemoryUtilizationHeapSys, 0),
			Set(CurrentMemoryUtilizationActive, 0),
			Set(MaxMemoryUtilizationAlloc, 0),
			Set(MaxMemoryUtilizationHeapSys, 0),
			Set(MaxMemoryUtilizationActive, 0),
		),
	)
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

func (h *Health) Share(fs ...func(*Health)) {
	for _, f := range fs {
		f(h)
	}
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

package health

import (
	"errors"
	"github.com/c9s/goprocinfo/linux"
	"runtime"
)

const (
	// General memory stats
	CurrentMemoryUtilizationAlloc     Stat = "CurrentMemoryUtilizationAlloc"
	CurrentMemoryUtilizationHeapSys   Stat = "CurrentMemoryUtilizationHeapSys"
	CurrentMemoryUtilizationActive    Stat = "CurrentMemoryUtilizationActive"
	MaxMemoryUtilizationAlloc         Stat = "MaxMemoryUtilizationAlloc"
	MaxMemoryUtilizationHeapSys       Stat = "MaxMemoryUtilizationHeapSys"
	MaxMemoryUtilizationActive        Stat = "MaxMemoryUtilizationActive"
	TotalRequestsReceived             Stat = "TotalRequestsReceived"
	TotalRequestsSuccessfullyServiced Stat = "TotalRequestsSuccessfullyServiced"
	TotalRequestsDenied               Stat = "TotalRequestsDenied"
)

var (
	// memoryStats are the health statistics dealing with memory usage.
	// these are automatically added to a Health monitor.
	memoryStats = []Option{
		CurrentMemoryUtilizationAlloc,
		CurrentMemoryUtilizationHeapSys,
		CurrentMemoryUtilizationActive,
		MaxMemoryUtilizationAlloc,
		MaxMemoryUtilizationHeapSys,
		MaxMemoryUtilizationActive,
	}

	// requestStats are the health statistics dealing with HTTP traffic.
	// these are automically added to a Health monitor.
	requestStats = []Option{
		TotalRequestsReceived,
		TotalRequestsSuccessfullyServiced,
		TotalRequestsDenied,
	}

	// Invalid stat option error
	ErrorInvalidOption = errors.New("Invalid stat option")
)

// Option describes an option that can be set on a Stats map.
// Various types implement this interface.
type Option interface {
	Set(Stats)
}

// Stat is a named piece of data to be tracked
type Stat string

// Create/Set the stat initially
func (s Stat) Set(stats Stats) {
	if _, ok := stats[s]; !ok {
		stats[s] = 0
	}
}

// HealthFunc functions are allowed to modify the passed-in stats.
type HealthFunc func(Stats)

func (f HealthFunc) Set(stats Stats) {
	f(stats)
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

// Stats is mapping of Stat to value
type Stats map[Stat]int

// NewStats constructs a Stats object preinitialized with the internal default
// statistics plus the given options.
func NewStats(options []Option) (s Stats) {
	s = make(Stats, len(memoryStats)+len(requestStats)+len(options))
	s.Apply(memoryStats)
	s.Apply(requestStats)
	s.Apply(options)
	return
}

func (s Stats) Set(stats Stats) {
	for key, value := range s {
		stats[key] = value
	}
}

// Clone returns a distinct copy of this Stats object
func (s Stats) Clone() Stats {
	clone := make(Stats, len(s))
	for key, value := range s {
		clone[key] = value
	}

	return clone
}

// Apply invokes each Option.Set() on this stats map.
func (s Stats) Apply(options []Option) {
	for _, option := range options {
		option.Set(s)
	}
}

// UpdateMemInfo takes memory information from a linux environment and
// sets the appropriate stats.
func (s Stats) UpdateMemInfo(memInfo *linux.MemInfo) {
	active := int(memInfo.Active * 1024)
	s[CurrentMemoryUtilizationActive] = active
	if active > s[MaxMemoryUtilizationActive] {
		s[MaxMemoryUtilizationActive] = active
	}
}

// UpdateMemStats takes a MemStats from the golang runtime and sets the
// appropriate stats.
func (s Stats) UpdateMemStats(memStats *runtime.MemStats) {
	alloc := int(memStats.Alloc)
	heapsys := int(memStats.HeapSys)

	// set current
	s[CurrentMemoryUtilizationAlloc] = alloc
	s[CurrentMemoryUtilizationHeapSys] = heapsys

	// set max
	if alloc > s[MaxMemoryUtilizationAlloc] {
		s[MaxMemoryUtilizationAlloc] = alloc
	}

	if heapsys > s[MaxMemoryUtilizationHeapSys] {
		s[MaxMemoryUtilizationHeapSys] = heapsys
	}
}

// UpdateMemory updates all the memory statistics
func (s Stats) UpdateMemory(memInfoReader *MemInfoReader) {
	memInfo, err := memInfoReader.Read()
	if err == nil {
		s.UpdateMemInfo(memInfo)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	s.UpdateMemStats(&memStats)
}

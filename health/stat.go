package health

import (
	"github.com/c9s/goprocinfo/linux"
	"runtime"
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

// Stat is a named piece of data to be tracked
type Stat string

// HealthFunc functions are allowed to modify the passed-in stats.
type HealthFunc func(Stats)

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

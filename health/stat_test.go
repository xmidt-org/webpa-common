package health

import (
	"github.com/c9s/goprocinfo/linux"
	"reflect"
	"runtime"
	"testing"
)

func TestClone(t *testing.T) {
	initial := Stats{
		CurrentMemoryUtilizationHeapSys: 123,
	}

	cloned := initial.Clone()
	if !reflect.DeepEqual(initial, cloned) {
		t.Errorf("Expected %v, got %v", initial, cloned)
	}

	cloned[CurrentMemoryUtilizationActive] = 123211
	if reflect.DeepEqual(initial, cloned) {
		t.Error("Clone should be a distinct instance")
	}
}

func TestApply(t *testing.T) {
	var testData = []struct {
		options  []Option
		initial  Stats
		expected Stats
	}{
		{
			options: []Option{Inc(CurrentMemoryUtilizationAlloc, 1)},
			initial: Stats{},
			expected: Stats{
				CurrentMemoryUtilizationAlloc: 1,
			},
		},
		{
			options: []Option{
				CurrentMemoryUtilizationAlloc,
				MaxMemoryUtilizationActive,
			},
			initial: Stats{},
			expected: Stats{
				CurrentMemoryUtilizationAlloc: 0,
				MaxMemoryUtilizationActive:    0,
			},
		},
		{
			options: []Option{
				CurrentMemoryUtilizationAlloc,
				MaxMemoryUtilizationActive,
			},
			initial: Stats{
				CurrentMemoryUtilizationAlloc: 12301,
			},
			expected: Stats{
				CurrentMemoryUtilizationAlloc: 12301,
				MaxMemoryUtilizationActive:    0,
			},
		},
		{
			options: []Option{
				Stats{
					CurrentMemoryUtilizationAlloc: 123,
					MaxMemoryUtilizationActive:    -982374,
				},
			},
			initial: Stats{},
			expected: Stats{
				CurrentMemoryUtilizationAlloc: 123,
				MaxMemoryUtilizationActive:    -982374,
			},
		},
		{
			options: []Option{
				Stats{
					CurrentMemoryUtilizationAlloc: 123,
					MaxMemoryUtilizationActive:    -982374,
				},
			},
			initial: Stats{
				MaxMemoryUtilizationAlloc: 56,
			},
			expected: Stats{
				MaxMemoryUtilizationAlloc:     56,
				CurrentMemoryUtilizationAlloc: 123,
				MaxMemoryUtilizationActive:    -982374,
			},
		},
	}

	for _, record := range testData {
		actual := record.initial.Clone()
		actual.Apply(record.options...)
		if !reflect.DeepEqual(record.expected, actual) {
			t.Errorf("Expected %v, got %v", record.expected, actual)
		}
	}
}

func TestInc(t *testing.T) {
	var testData = []struct {
		stat      Stat
		increment int
		initial   Stats
		expected  Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			1,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 1},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			-12,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: -12},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			72,
			Stats{CurrentMemoryUtilizationHeapSys: 0},
			Stats{CurrentMemoryUtilizationHeapSys: 72},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			6,
			Stats{CurrentMemoryUtilizationHeapSys: 45},
			Stats{CurrentMemoryUtilizationHeapSys: 51},
		},
	}

	for _, record := range testData {
		Inc(record.stat, record.increment)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestSet(t *testing.T) {
	var testData = []struct {
		stat     Stat
		newValue int
		initial  Stats
		expected Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			123,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 123},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			37842,
			Stats{CurrentMemoryUtilizationHeapSys: 42734987},
			Stats{CurrentMemoryUtilizationHeapSys: 37842},
		},
	}

	for _, record := range testData {
		Set(record.stat, record.newValue)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestEnsure(t *testing.T) {
	var testData = []struct {
		stat     Stat
		initial  Stats
		expected Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 0},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			Stats{CurrentMemoryUtilizationHeapSys: -157},
			Stats{CurrentMemoryUtilizationHeapSys: -157},
		},
	}

	for _, record := range testData {
		Ensure(record.stat)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestUpdateMemInfo(t *testing.T) {
	var testData = []struct {
		memInfo  linux.MemInfo
		initial  Stats
		expected Stats
	}{
		// empty initial Stats
		{
			linux.MemInfo{
				Active: 3457,
			},
			Stats{},
			Stats{
				CurrentMemoryUtilizationActive: int(3457 * 1024),
				MaxMemoryUtilizationActive:     int(3457 * 1024),
			},
		},
		// max is less than current
		{
			linux.MemInfo{
				Active: 13,
			},
			Stats{
				MaxMemoryUtilizationActive: 1,
			},
			Stats{
				CurrentMemoryUtilizationActive: int(13 * 1024),
				MaxMemoryUtilizationActive:     int(13 * 1024),
			},
		},
		// max is larger than current
		{
			linux.MemInfo{
				Active: 271,
			},
			Stats{
				MaxMemoryUtilizationActive: int(34872 * 1024),
			},
			Stats{
				CurrentMemoryUtilizationActive: int(271 * 1024),
				MaxMemoryUtilizationActive:     int(34872 * 1024),
			},
		},
	}

	for _, record := range testData {
		actual := record.initial.Clone()
		actual.UpdateMemInfo(&record.memInfo)
		if !reflect.DeepEqual(record.expected, actual) {
			t.Errorf("Expected %v, but got %v", record.expected, actual)
		}
	}
}

func TestUpdateMemStats(t *testing.T) {
	var testData = []struct {
		memStats runtime.MemStats
		initial  Stats
		expected Stats
	}{
		// empty initial Stats
		{
			runtime.MemStats{
				Alloc:   247,
				HeapSys: 2381,
			},
			Stats{},
			Stats{
				CurrentMemoryUtilizationAlloc:   247,
				MaxMemoryUtilizationAlloc:       247,
				CurrentMemoryUtilizationHeapSys: 2381,
				MaxMemoryUtilizationHeapSys:     2381,
			},
		},
		// current is less than max
		{
			runtime.MemStats{
				Alloc:   3874,
				HeapSys: 1234,
			},
			Stats{
				CurrentMemoryUtilizationAlloc:   12354,
				MaxMemoryUtilizationAlloc:       927412,
				CurrentMemoryUtilizationHeapSys: 7897,
				MaxMemoryUtilizationHeapSys:     827123,
			},
			Stats{
				CurrentMemoryUtilizationAlloc:   3874,
				MaxMemoryUtilizationAlloc:       927412,
				CurrentMemoryUtilizationHeapSys: 1234,
				MaxMemoryUtilizationHeapSys:     827123,
			},
		},
		// current is greater than max
		{
			runtime.MemStats{
				Alloc:   8742,
				HeapSys: 2903209,
			},
			Stats{
				CurrentMemoryUtilizationAlloc:   135,
				MaxMemoryUtilizationAlloc:       1254,
				CurrentMemoryUtilizationHeapSys: 5412,
				MaxMemoryUtilizationHeapSys:     12345,
			},
			Stats{
				CurrentMemoryUtilizationAlloc:   8742,
				MaxMemoryUtilizationAlloc:       8742,
				CurrentMemoryUtilizationHeapSys: 2903209,
				MaxMemoryUtilizationHeapSys:     2903209,
			},
		},
	}

	for _, record := range testData {
		actual := record.initial.Clone()
		actual.UpdateMemStats(&record.memStats)
		if !reflect.DeepEqual(record.expected, actual) {
			t.Errorf("Expected %v, but got %v", record.expected, actual)
		}
	}
}

func TestUpdateMemory(t *testing.T) {
	memInfoReader := &MemInfoReader{"meminfo.test"}
	stats := make(Stats)
	stats.UpdateMemory(memInfoReader)

	// each key in commonStats should be present in the output
	for key, _ := range commonStats {
		if _, ok := stats[key]; !ok {
			t.Errorf("Key %s not present in ServeHTTP results", key)
		}
	}
}

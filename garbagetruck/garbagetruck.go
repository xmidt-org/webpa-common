package garbagetruck

import (
	"runtime/debug"
	"sync"
	"time"
)

// GarbageTruck contains all the information for running it.
type GarbageTruck struct {
	interval time.Duration
	log      Logger
}

// Logger interface for GarbageTruck
type Logger interface {
	Debug(params ...interface{})
	Error(params ...interface{})
}

// SetInterval sets the time a which the ticker will tick.
func (gt *GarbageTruck) SetInterval(t time.Duration) { gt.interval = t }

// SetLog sets the logger.
func (gt *GarbageTruck) SetLog(lg Logger) { gt.log = lg }

// New creates new GarbageTruck and starts it.
func New(t time.Duration, lg Logger, wg *sync.WaitGroup, shutdown <-chan struct{}) *GarbageTruck {
	gt := new(GarbageTruck)
	gt.SetInterval(t)
	gt.SetLog(lg)

	gt.Run(wg, shutdown)

	return gt
}

// Start the GarbageTruck.  Logs the garbage collection stats.
func (gt *GarbageTruck) Run(wg *sync.WaitGroup, shutdown <-chan struct{}) error {
	gt.log.Debug("Garbage Truck Started")

	wg.Add(1)
	go func() {
		ticker := time.NewTicker(gt.interval)

		defer wg.Done()
		defer gt.log.Debug("Garbage Truck Stopped")
		defer ticker.Stop()

		gcStats := new(debug.GCStats)
		for {
			select {
			case <-shutdown:
				return
			case <-ticker.C:
				debug.FreeOSMemory()
				debug.ReadGCStats(gcStats)
				gt.log.Error("garbage collection stats, LastGC: %v, NumGC: %v, PauseTotal: %v", gcStats.LastGC, gcStats.NumGC, gcStats.PauseTotal)
			}
		}
	}()
	
	return nil
}

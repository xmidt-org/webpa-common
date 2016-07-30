package garbagetruck

import (
	"github.com/ian-kent/go-log/logger"
	"runtime/debug"
	"sync"
	"time"
)

// GarbageTruck contains all the information for running it.
type GarbageTruck struct {
	interval time.Duration
	log      logger.Logger
	wg       *sync.WaitGroup
	stop     chan bool
}

// SetInterval sets the time a which the ticker will tick.
func (gt *GarbageTruck) SetInterval(t time.Duration) {gt.interval = t}

// SetLog sets the logger.
func (gt *GarbageTruck) SetLog(lg logger.Logger) {gt.log = lg}

// SetWaitGroup sets a sync.WaitGroup.
func (gt *GarbageTruck) SetWaitGroup(wg *sync.WaitGroup) {gt.wg = wg}

// New creates new GarbageTruck and starts it.
func New(t time.Duration, lg logger.Logger, wg *sync.WaitGroup) *GarbageTruck {
	gt := new(GarbageTruck)
	gt.SetInterval(t)
	gt.SetLog(lg)
	gt.SetWaitGroup(wg)
	gt.stop = make( chan bool, 1 )
	
	gt.Start()
	
	return gt
}

// Stop the GarbageTruck
func (gt *GarbageTruck) Stop() {
	close(gt.stop)
}

// Start the GarbageTruck.  Logs the garbage collection stats.
func (gt *GarbageTruck) Start() {
	gt.log.Trace("Garbage Truck Started")
	
	gt.wg.Add(1)
	go func() {
		ticker := time.NewTicker(gt.interval)
		
		defer gt.wg.Done()
		defer gt.log.Trace("Garbage Truck Stopped")
		defer ticker.Stop()
		
		gcStats := new(debug.GCStats)
		for {
			select {
				case _, ok := <- gt.stop:
					if !ok {
						return
					} 
				case <- ticker.C:
					debug.FreeOSMemory()
					debug.ReadGCStats(gcStats)
					gt.log.Error("garbage collection stats, LastGC: %v, NumGC: %v, PauseTotal: %v", gcStats.LastGC, gcStats.NumGC, gcStats.PauseTotal)
			}
		}
	}()
}
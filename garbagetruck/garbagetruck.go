package garbagetruck

import (
	"github.com/ian-kent/go-log/logger"
	"runtime/debug"
	"sync"
	"time"
)

type GarbageTruck struct {
	interval time.Duration
	log      logger.Logger
	wg       *sync.WaitGroup
	stop     chan bool
}

func (gt *GarbageTruck) SetInterval(t time.Duration) {gt.interval = t}
func (gt *GarbageTruck) SetLog(lg logger.Logger) {gt.log = lg}
func (gt *GarbageTruck) SetWaitGroup(wg *sync.WaitGroup) {gt.wg = wg}

func New(t time.Duration, lg logger.Logger, wg *sync.WaitGroup) *GarbageTruck {
	gt := new(GarbageTruck)
	gt.SetInterval(t)
	gt.SetLog(lg)
	gt.SetWaitGroup(wg)
	
	return gt
}

func (gt *GarbageTruck) Stop() {
	close(gt.stop)
}

func (gt *GarbageTruck) Start() {
	gt.log.Trace("Garbage Truck Started")
	
	gt.wg.Add(1)
	go func() {
		gcStats := new(debug.GCStats)
		ticker := time.NewTicker(gt.interval)
		
		defer ticker.Stop()
		defer gt.log.Trace("Garbage Truck Stopped")
		defer gt.wg.Done()
		
		for {
			select {
				case _, ok := <- gt.stop:
					if !ok {
						return
					} 
				case <- ticker.C:
					debug.FreeOSMemory()
					debug.ReadGCStats(gcStats)
					gt.log.Error("garbage collection stats LastGC: %v, NumGC: %v, PauseTotal: %v", gcStats.LastGC, gcStats.NumGC, gcStats.PauseTotal)
			}
		}
	}()
}
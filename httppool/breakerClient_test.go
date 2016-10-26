package httppool

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestDispatchWithBreaker(t *testing.T) {
	wg := &sync.WaitGroup{}
	
	timeout := time.Second * 10
	threshold := int64(3)
	client := &http.Client{
		Transport: &http.Transport{},
		Timeout: timeout,
	}	

	dispatcher := (&Client{
		Workers: 5,
		QueueSize: 100,
		Logger: testLogger,
		Handler: BreakerClient(timeout, threshold, testLogger, client),
	}).Start()
	
	url := "http://www.google.com/"
	req, _ := http.NewRequest("GET", url, nil)

	consumer := func(u string, wg *sync.WaitGroup) Consumer {
		return func(resp *http.Response, req *http.Request) {
			defer wg.Done()
			
			if resp == nil {
				t.Error("Failed to obtain a response.  url: %v", u)
			}
		}
	}(url, wg)
	
	task := RequestTask(req, consumer)
	if taken, err := dispatcher.Offer(task); err != nil {
		t.Error("offer dropped. taken: %v\n", taken)
	} else {
		wg.Add(1)
	}
	
	wg.Wait()
}
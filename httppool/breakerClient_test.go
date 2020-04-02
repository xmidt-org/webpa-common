package httppool

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/xmidt-org/webpa-common/logging"
)

func TestDispatchWithBreaker(t *testing.T) {
	var (
		wg = new(sync.WaitGroup)

		logger    = logging.DefaultLogger()
		timeout   = time.Second * 10
		threshold = int64(3)
		client    = &http.Client{
			Transport: &http.Transport{},
			Timeout:   timeout,
		}

		dispatcher = (&Client{
			Workers:   5,
			QueueSize: 100,
			Logger:    logger,
			Handler:   BreakerClient(timeout, threshold, logger, client),
		}).Start()

		url = "http://www.google.com/"
		req = httptest.NewRequest("GET", url, nil)

		consumer = func(u string, wg *sync.WaitGroup) Consumer {
			return func(resp *http.Response, req *http.Request) {
				defer wg.Done()

				if resp == nil {
					t.Errorf("Failed to obtain a response.  url: %v", u)
				}
			}
		}(url, wg)

		task = RequestTask(req, consumer)
	)

	if taken, err := dispatcher.Offer(task); err != nil {
		t.Fatalf("offer dropped. taken: %v\n", taken)
	}

	wg.Wait()
}

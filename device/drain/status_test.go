package drain

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testStatus(t *testing.T, active bool, j Job, p Progress, expectedJSON string) {
	var (
		assert = assert.New(t)

		d      = new(mockDrainer)
		status = Status{d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	d.On("Status").Return(active, j, p).Once()
	status.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.JSONEq(expectedJSON, response.Body.String())
	d.AssertExpectations(t)
}

func TestStatus(t *testing.T) {
	var (
		zeroTime = time.Time{}.Format(time.RFC3339Nano)
		now      = time.Now()

		testData = []struct {
			active       bool
			j            Job
			p            Progress
			expectedJSON string
		}{
			{
				// when no job has been run since the server started:
				false,
				Job{},
				Progress{},
				fmt.Sprintf(`{"active": false, "job": {"count": 0, "filter": {"key":"", "values":null}}, "progress": {"visited": 0, "drained": 0, "skipped": 0, "started": "%s"}}`, zeroTime),
			},

			{
				true,
				Job{Count: 67283, Percent: 97, Rate: 127, Tick: 17 * time.Second},
				Progress{Visited: 12, Drained: 4, Skipped: 0, Started: now, Finished: &now},
				fmt.Sprintf(`{"active": true, "job": {"count": 67283, "percent": 97, "rate": 127, "tick": "17s", "filter": {"key":"", "values":null}}, "progress": {"visited": 12, "drained": 4, "skipped": 0, "started": "%s", "finished": "%s"}}`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)),
			},
		}
	)

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			testStatus(t, record.active, record.j, record.p, record.expectedJSON)
		})
	}
}

// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package drain

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/device/devicegate"
)

func testStatus(t *testing.T, active bool, j Job, p Progress, expectedJSON string) {
	var (
		assert = assert.New(t)

		d      = new(mockDrainer)
		status = Status{d}

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	// nolint: typecheck
	d.On("Status").Return(active, j, p).Once()
	status.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)
	assert.JSONEq(expectedJSON, response.Body.String())
	// nolint: typecheck
	d.AssertExpectations(t)
}

func TestStatus(t *testing.T) {
	var (
		zeroTime = time.Time{}.Format(time.RFC3339Nano)
		now      = time.Now()

		df = &drainFilter{
			filter: &devicegate.FilterGate{
				FilterStore: devicegate.FilterStore(map[string]devicegate.Set{
					"test": &devicegate.FilterSet{Set: map[interface{}]bool{
						"test1": true,
						"test2": true,
					}},
				}),
			},
			filterRequest: devicegate.FilterRequest{
				Key:    "test",
				Values: []interface{}{"test1", "test2"},
			},
		}

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
				fmt.Sprintf(`{"active": false, "job": {"count": 0}, "progress": {"visited": 0, "drained": 0, "started": "%s"}}`, zeroTime),
			},

			{
				true,
				Job{Count: 67283, Percent: 97, Rate: 127, Tick: 17 * time.Second},
				Progress{Visited: 12, Drained: 4, Started: now, Finished: &now},
				fmt.Sprintf(`{"active": true, "job": {"count": 67283, "percent": 97, "rate": 127, "tick": "17s"}, "progress": {"visited": 12, "drained": 4, "started": "%s", "finished": "%s"}}`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)),
			},
			{
				true,
				Job{Count: 67283, Percent: 97, Rate: 127, Tick: 17 * time.Second, DrainFilter: df},
				Progress{Visited: 12, Drained: 4, Started: now, Finished: &now},
				fmt.Sprintf(`{"active": true, "job": {"count": 67283, "percent": 97, "rate": 127, "tick": "17s", "filter": {"key":"test", "values":["test1", "test2"]}}, "progress": {"visited": 12, "drained": 4, "started": "%s", "finished": "%s"}}`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)),
			},
		}
	)

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			testStatus(t, record.active, record.j, record.p, record.expectedJSON)
		})
	}
}

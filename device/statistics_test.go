package device

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const EqualityThreshold = 1000

func Abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
func almostEqual(a, b time.Duration) bool {
	return Abs(a.Nanoseconds()-b.Nanoseconds()) <= EqualityThreshold
}

func testStatisticsInitialStateDefaultNow(t *testing.T) {
	var (
		assert              = assert.New(t)
		require             = require.New(t)
		expectedConnectedAt = time.Now()

		statistics = NewStatistics(
			nil,
			expectedConnectedAt,
		)
	)

	assert.Zero(statistics.BytesSent())
	assert.Zero(statistics.BytesReceived())
	assert.Zero(statistics.MessagesSent())
	assert.Zero(statistics.MessagesReceived())
	assert.Zero(statistics.Duplications())
	assert.Equal(expectedConnectedAt.UTC(), statistics.ConnectedAt())
	expectedUpTime := time.Now().Sub(expectedConnectedAt)
	uptime := statistics.UpTime()
	assert.True(almostEqual(expectedUpTime, uptime),
		"TimeSince Connected %dns,  UpTime %dns with a margin of %d, actual %d",
		expectedUpTime.Nanoseconds(), uptime.Nanoseconds(), EqualityThreshold, Abs(expectedUpTime.Nanoseconds()-uptime.Nanoseconds()))

	data, err := statistics.MarshalJSON()
	require.NotEmpty(data)
	require.NoError(err)

	var actualJSON map[string]interface{}
	require.NoError(json.Unmarshal(data, &actualJSON))
	assert.Equal(float64(0), actualJSON["bytesSent"])
	assert.Equal(float64(0), actualJSON["messagesSent"])
	assert.Equal(float64(0), actualJSON["bytesReceived"])
	assert.Equal(float64(0), actualJSON["messagesReceived"])
	assert.Equal(float64(0), actualJSON["duplications"])

	actualConnectedAt, err := time.Parse(time.RFC3339Nano, actualJSON["connectedAt"].(string))
	require.NoError(err)
	assert.True(
		actualConnectedAt.UTC().Equal(expectedConnectedAt.UTC()) || actualConnectedAt.UTC().After(expectedConnectedAt.UTC()),
		"%s must be greater than or equal to %s",
		actualConnectedAt.UTC(),
		expectedConnectedAt.UTC(),
	)

	actualUpTime, err := time.ParseDuration(actualJSON["upTime"].(string))
	require.NoError(err)
	assert.True(actualUpTime >= 0)
}

func testStatisticsInitialStateCustomNow(t *testing.T) {
	var (
		assert              = assert.New(t)
		require             = require.New(t)
		expectedConnectedAt = time.Now()
		expectedUpTime      = 149 * time.Hour

		statistics = NewStatistics(
			func() time.Time {
				return expectedConnectedAt.Add(expectedUpTime)
			},
			expectedConnectedAt,
		)
	)

	assert.Zero(statistics.BytesSent())
	assert.Zero(statistics.BytesReceived())
	assert.Zero(statistics.MessagesSent())
	assert.Zero(statistics.MessagesReceived())
	assert.Zero(statistics.Duplications())
	assert.Equal(expectedConnectedAt.UTC(), statistics.ConnectedAt())
	assert.Equal(expectedUpTime, statistics.UpTime())

	data, err := statistics.MarshalJSON()
	require.NotEmpty(data)
	require.NoError(err)

	assert.JSONEq(
		fmt.Sprintf(
			`{"duplications": 0, "bytesSent": 0, "messagesSent": 0, "bytesReceived": 0, "messagesReceived": 0, "connectedAt": "%s", "upTime": "%s"}`,
			expectedConnectedAt.UTC().Format(time.RFC3339Nano),
			expectedUpTime,
		),
		string(data),
	)
}

func testStatisticsConcurrency(t *testing.T) {
	var (
		assert              = assert.New(t)
		require             = require.New(t)
		expectedConnectedAt = time.Now()
		expectedUpTime      = 459234 * time.Second

		values = []int{17, -8, 124, 12, 1900, -3, 15}
		gate   = new(sync.WaitGroup)
		done   = new(sync.WaitGroup)

		statistics = NewStatistics(
			func() time.Time {
				return expectedConnectedAt.Add(expectedUpTime)
			},
			expectedConnectedAt,
		)
	)

	gate.Add(1)
	done.Add(len(values))
	expectedValue := 0
	for _, v := range values {
		expectedValue += v
		go func(v int) {
			defer done.Done()
			gate.Wait()

			statistics.AddBytesSent(v)
			statistics.AddMessagesSent(v)
			statistics.AddBytesReceived(v)
			statistics.AddMessagesReceived(v)
			statistics.AddDuplications(v)
		}(v)
	}

	gate.Done()
	done.Wait()

	assert.Equal(expectedValue, statistics.BytesSent())
	assert.Equal(expectedValue, statistics.MessagesSent())
	assert.Equal(expectedValue, statistics.BytesReceived())
	assert.Equal(expectedValue, statistics.MessagesReceived())
	assert.Equal(expectedValue, statistics.Duplications())
	assert.Equal(expectedConnectedAt.UTC(), statistics.ConnectedAt())
	assert.Equal(expectedUpTime, statistics.UpTime())

	data, err := statistics.MarshalJSON()
	require.NotEmpty(data)
	require.NoError(err)

	assert.JSONEq(
		fmt.Sprintf(
			`{"duplications": %d, "bytesSent": %d, "messagesSent": %d, "bytesReceived": %d, "messagesReceived": %d, "connectedAt": "%s", "upTime": "%s"}`,
			expectedValue,
			expectedValue,
			expectedValue,
			expectedValue,
			expectedValue,
			expectedConnectedAt.UTC().Format(time.RFC3339Nano),
			expectedUpTime,
		),
		string(data),
	)
}

func TestStatistics(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		t.Run("DefaultNow", testStatisticsInitialStateDefaultNow)
		t.Run("CustomNow", testStatisticsInitialStateCustomNow)
	})

	t.Run("Concurrency", testStatisticsConcurrency)
}

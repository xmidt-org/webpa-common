package device

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testStatisticsInitialStateDefaultNow(t *testing.T) {
	var (
		assert              = assert.New(t)
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
	assert.True(time.Now().Sub(expectedConnectedAt) <= statistics.UpTime())
}

func testStatisticsInitialStateCustomNow(t *testing.T) {
	var (
		assert              = assert.New(t)
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
}

func TestStatistics(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		t.Run("DefaultNow", testStatisticsInitialStateDefaultNow)
		t.Run("CustomNow", testStatisticsInitialStateCustomNow)
	})
}

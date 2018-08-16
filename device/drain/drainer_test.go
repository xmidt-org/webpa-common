package drain

import (
	"fmt"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewNoRegistry(t *testing.T) {
	var (
		assert  = assert.New(t)
		manager = generateManager(assert, 0)
	)

	assert.Panics(func() {
		New(WithConnector(manager))
	})
}

func testNewNoConnector(t *testing.T) {
	var (
		assert  = assert.New(t)
		manager = generateManager(assert, 0)
	)

	assert.Panics(func() {
		New(WithRegistry(manager))
	})
}

func TestNew(t *testing.T) {
	t.Run("NoRegistry", testNewNoRegistry)
	t.Run("NoConnector", testNewNoConnector)
}

func testDrainerDrainAll(t *testing.T, deviceCount int) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, uint64(deviceCount))

		firstTime        = true
		expectedStarted  = time.Now()
		expectedFinished = expectedStarted.Add(10 * time.Minute)

		stopCalled = false
		stop       = func() {
			stopCalled = true
		}

		ticker = make(chan time.Time, 1)

		d = New(
			WithLogger(logger),
			WithRegistry(manager),
			WithConnector(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	require.NotNil(d)
	d.(*drainer).now = func() time.Time {
		if firstTime {
			firstTime = false
			return expectedStarted
		}

		return expectedFinished
	}

	d.(*drainer).newTicker = func(d time.Duration) (<-chan time.Time, func()) {
		assert.Equal(time.Second, d)
		return ticker, stop
	}

	defer d.Cancel() // cleanup in case of horribleness

	done, err := d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress := d.Status()
	assert.False(active)
	assert.Equal(Job{}, job)
	assert.Equal(Progress{}, progress)

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	done, err = d.Start(Job{Rate: 100, Tick: time.Second})
	require.NoError(err)
	require.NotNil(done)

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	active, job, progress = d.Status()
	assert.True(active)
	assert.Equal(Job{Count: deviceCount, Rate: 100, Tick: time.Second}, job)
	assert.Equal(Progress{Visited: 0, Drained: 0, Started: expectedStarted.UTC(), Finished: nil}, progress)

	go func() {
		ticks := deviceCount / 100
		if (deviceCount % 100) > 0 {
			ticks++
		}

		for i := 0; i < ticks; i++ {
			ticker <- time.Time{}
		}
	}()

	close(manager.pauseVisit)
	select {
	case <-done:
		// passed
	case <-time.After(5 * time.Second):
		assert.Fail("Drain failed to complete")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(float64(deviceCount)))

	done, err = d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress = d.Status()
	assert.False(active)
	assert.Equal(Job{Count: deviceCount, Rate: 100, Tick: time.Second}, job)
	assert.Equal(deviceCount, progress.Visited)
	assert.Equal(deviceCount, progress.Drained)
	assert.Equal(expectedStarted.UTC(), progress.Started)
	require.NotNil(progress.Finished)
	assert.Equal(expectedFinished.UTC(), *progress.Finished)

	assert.Empty(manager.devices)
}

func testDrainerDisconnectAll(t *testing.T, deviceCount int) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, uint64(deviceCount))

		firstTime        = true
		expectedStarted  = time.Now()
		expectedFinished = expectedStarted.Add(10 * time.Minute)

		d = New(
			WithLogger(logger),
			WithRegistry(manager),
			WithConnector(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	require.NotNil(d)
	d.(*drainer).now = func() time.Time {
		if firstTime {
			firstTime = false
			return expectedStarted
		}

		return expectedFinished
	}

	defer d.Cancel() // cleanup in case of panic

	done, err := d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress := d.Status()
	assert.False(active)
	assert.Equal(Job{}, job)
	assert.Equal(Progress{}, progress)

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	done, err = d.Start(Job{})
	require.NoError(err)
	require.NotNil(done)

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	active, job, progress = d.Status()
	assert.True(active)
	assert.Equal(Job{Count: deviceCount}, job)
	assert.Equal(Progress{Visited: 0, Drained: 0, Started: expectedStarted.UTC(), Finished: nil}, progress)

	close(manager.pauseVisit)
	select {
	case <-done:
		// passed
	case <-time.After(5 * time.Second):
		assert.Fail("Disconnect all failed to complete")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(float64(deviceCount)))

	done, err = d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress = d.Status()
	assert.False(active)
	assert.Equal(Job{Count: deviceCount}, job)
	assert.Equal(deviceCount, progress.Visited)
	assert.Equal(deviceCount, progress.Drained)
	assert.Equal(expectedStarted.UTC(), progress.Started)
	require.NotNil(progress.Finished)
	assert.Equal(expectedFinished.UTC(), *progress.Finished)

	assert.Empty(manager.devices)
}

func TestDrainer(t *testing.T) {
	deviceCounts := []int{0, 1, 2, disconnectBatchSize - 1, disconnectBatchSize, disconnectBatchSize + 1, 1709}

	t.Run("DisconnectAll", func(t *testing.T) {
		for _, deviceCount := range deviceCounts {
			t.Run(fmt.Sprintf("deviceCount=%d", deviceCount), func(t *testing.T) {
				testDrainerDisconnectAll(t, deviceCount)
			})
		}
	})

	t.Run("DrainAll", func(t *testing.T) {
		for _, deviceCount := range deviceCounts {
			t.Run(fmt.Sprintf("deviceCount=%d", deviceCount), func(t *testing.T) {
				testDrainerDrainAll(t, deviceCount)
			})
		}
	})
}

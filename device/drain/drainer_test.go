package drain

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/device/devicegate"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
)

type deviceInfo struct {
	claims map[string]interface{}
	count  int
}

func testJobNormalize(t *testing.T) {
	testDrainFilter := &drainFilter{
		filter: &devicegate.FilterGate{
			FilterStore: devicegate.FilterStore(map[string]devicegate.Set{
				"test": devicegate.FilterSet(map[interface{}]bool{
					"testValue":  true,
					"testValue2": true,
				}),
			}),
		},
		filterRequest: devicegate.FilterRequest{
			Key:    "test",
			Values: []interface{}{"testValue", "testValue2"},
		},
	}

	testData := []struct {
		deviceCount int
		actual      Job
		expected    Job
	}{
		{1000, Job{}, Job{Count: 1000}},
		{972, Job{Count: -1, Rate: -1}, Job{Count: 972}},
		{1873, Job{Rate: 52}, Job{Count: 1873, Rate: 52, Tick: time.Second}},
		{438742, Job{Tick: 15 * time.Minute}, Job{Count: 438742}},
		{0, Job{Percent: 0}, Job{Count: 0}},
		{123752, Job{Percent: 17}, Job{Count: 21037, Percent: 17}},
		{73, Job{Percent: 100}, Job{Count: 73, Percent: 100}},
		{90, Job{DrainFilter: testDrainFilter}, Job{Count: 90, DrainFilter: testDrainFilter}},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				actual = record.actual
			)

			actual.normalize(record.deviceCount)
			assert.Equal(record.expected, actual)
		})
	}
}

func TestJob(t *testing.T) {
	t.Run("Normalize", testJobNormalize)
}

func testWithLoggerDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		d      = new(drainer)
	)

	WithLogger(nil)(d)
	assert.NotNil(d.logger)
}

func testWithLoggerCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)
		d      = new(drainer)
	)

	WithLogger(logger)(d)
	assert.Equal(logger, d.logger)
}

func TestWithLogger(t *testing.T) {
	t.Run("Default", testWithLoggerDefault)
	t.Run("Custom", testWithLoggerCustom)
}

func testWithRegistryNil(t *testing.T) {
	assert.Panics(t, func() {
		WithRegistry(nil)
	})
}

func testWithRegistryCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		d       = new(drainer)
		manager = new(stubManager)
	)

	WithRegistry(manager)(d)
	assert.Equal(manager, d.registry)
}

func TestWithRegistry(t *testing.T) {
	t.Run("Nil", testWithRegistryNil)
	t.Run("Custom", testWithRegistryCustom)
}

func testWithConnectorNil(t *testing.T) {
	assert.Panics(t, func() {
		WithConnector(nil)
	})
}

func testWithConnectorCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		d       = new(drainer)
		manager = new(stubManager)
	)

	WithConnector(manager)(d)
	assert.Equal(manager, d.connector)
}

func TestWithConnector(t *testing.T) {
	t.Run("Nil", testWithConnectorNil)
	t.Run("Custom", testWithConnectorCustom)
}

func testWithManagerNil(t *testing.T) {
	assert.Panics(t, func() {
		WithManager(nil)
	})
}

func testWithManagerCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		d       = new(drainer)
		manager = new(stubManager)
	)

	WithManager(manager)(d)
	assert.Equal(manager, d.registry)
	assert.Equal(manager, d.connector)
}

func TestWithManager(t *testing.T) {
	t.Run("Nil", testWithManagerNil)
	t.Run("Custom", testWithManagerCustom)
}

func testWithStateGaugeDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		d      = new(drainer)
	)

	WithStateGauge(nil)(d)
	assert.NotNil(d.m.state)
}

func testWithStateGaugeCustom(t *testing.T) {
	var (
		assert   = assert.New(t)
		d        = new(drainer)
		provider = xmetricstest.NewProvider(nil)
		gauge    = provider.NewGauge("test")
	)

	WithStateGauge(gauge)(d)
	assert.Equal(gauge, d.m.state)
}

func TestWithStateGauge(t *testing.T) {
	t.Run("Default", testWithStateGaugeDefault)
	t.Run("Custom", testWithStateGaugeCustom)
}

func testWithDrainCounterDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		d      = new(drainer)
	)

	WithDrainCounter(nil)(d)
	assert.NotNil(d.m.counter)
}

func testWithDrainCounterCustom(t *testing.T) {
	var (
		assert   = assert.New(t)
		d        = new(drainer)
		provider = xmetricstest.NewProvider(nil)
		counter  = provider.NewCounter("test")
	)

	WithDrainCounter(counter)(d)
	assert.Equal(counter, d.m.counter)
}

func TestWithDrainCounter(t *testing.T) {
	t.Run("Default", testWithDrainCounterDefault)
	t.Run("Custom", testWithDrainCounterCustom)
}

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

	done, job, err = d.Start(Job{Rate: 100, Tick: time.Second})
	require.NoError(err)
	require.NotNil(done)
	assert.Equal(Job{Count: deviceCount, Rate: 100, Tick: time.Second}, job)

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	{
		done, job, err := d.Start(Job{Rate: 123, Tick: time.Minute})
		assert.Nil(done)
		assert.Error(err)
		assert.Equal(Job{}, job)
	}

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

	close(manager.pauseDisconnect)
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
	assert.True(stopCalled)
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

	done, job, err = d.Start(Job{})
	require.NoError(err)
	require.NotNil(done)
	assert.Equal(Job{Count: deviceCount}, job)

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	{
		done, job, err := d.Start(Job{Rate: 123, Tick: time.Minute})
		assert.Nil(done)
		assert.Error(err)
		assert.Equal(Job{}, job)
	}

	active, job, progress = d.Status()
	assert.True(active)
	assert.Equal(Job{Count: deviceCount}, job)
	assert.Equal(Progress{Visited: 0, Drained: 0, Started: expectedStarted.UTC(), Finished: nil}, progress)

	close(manager.pauseDisconnect)
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

func testDrainerVisitCancel(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, 100)

		d = New(
			WithLogger(logger),
			WithManager(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	require.NotNil(d)
	d.Start(Job{})
	done, err := d.Cancel()
	require.NoError(err)
	require.NotNil(done)
	close(manager.pauseVisit)

	select {
	case <-done:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("The job did not complete after being canceled")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))
}

func testDrainerDisconnectCancel(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, 100)

		d = New(
			WithLogger(logger),
			WithManager(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	require.NotNil(d)
	defer d.Cancel()
	d.Start(Job{})
	close(manager.pauseVisit)

	select {
	case <-manager.disconnect:
	case <-time.After(5 * time.Second):
		assert.Fail("Disconnect was not called")
		return
	}

	done, err := d.Cancel()
	require.NoError(err)
	require.NotNil(done)
	close(manager.pauseDisconnect)

	select {
	case <-done:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("The job did not complete after being canceled")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Minimum(1.0))
}

func testDrainerDrainCancel(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		manager = generateManager(assert, 100)

		stopCalled = false
		stop       = func() {
			stopCalled = true
		}
		ticker = make(chan time.Time, 1)

		d = New(
			WithLogger(logger),
			WithManager(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	require.NotNil(d)
	defer d.Cancel()

	d.(*drainer).newTicker = func(d time.Duration) (<-chan time.Time, func()) {
		assert.Equal(time.Second, d)
		return ticker, stop
	}

	done, job, err := d.Start(Job{Percent: 20, Rate: 5})
	require.NoError(err)
	require.NotNil(done)
	assert.Equal(
		Job{Count: 20, Percent: 20, Rate: 5, Tick: time.Second},
		job,
	)

	active, job, _ := d.Status()
	assert.True(active)
	assert.Equal(
		Job{Count: 20, Percent: 20, Rate: 5, Tick: time.Second},
		job,
	)

	done, err = d.Cancel()
	require.NotNil(done)
	require.NoError(err)
	ticker <- time.Time{}
	close(manager.pauseVisit)
	close(manager.pauseDisconnect)

	select {
	case <-done:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Drain failed to complete")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Minimum(0.0))

	assert.True(stopCalled)
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

	t.Run("VisitCancel", testDrainerVisitCancel)
	t.Run("DisconnectCancel", testDrainerDisconnectCancel)
	t.Run("DrainCancel", testDrainerDrainCancel)
}

func testDrainFilter(t *testing.T, deviceTypeOne deviceInfo, deviceTypeTwo deviceInfo, df DrainFilter, expectedSkipped int, count int) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		// generate manager with devices that have two different metadatas
		manager = generateManagerWithDifferentDevices(assert, deviceTypeOne.claims, uint64(deviceTypeOne.count), deviceTypeTwo.claims, uint64(deviceTypeTwo.count))

		firstTime        = true
		expectedStarted  = time.Now()
		expectedFinished = expectedStarted.Add(10 * time.Minute)

		stopCalled = false
		stop       = func() {
			stopCalled = true
		}

		ticker     = make(chan time.Time, 1)
		totalCount = deviceTypeOne.count + deviceTypeTwo.count
		realCount  = totalCount

		d = New(
			WithLogger(logger),
			WithRegistry(manager),
			WithConnector(manager),
			WithStateGauge(provider.NewGauge("state")),
			WithDrainCounter(provider.NewCounter("counter")),
		)
	)

	if count > 0 {
		realCount = count
	}

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

	// test that cancel will error if there is not a drain job in progress
	done, err := d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	// test status when drain hasn't started
	active, job, progress := d.Status()
	assert.False(active)
	assert.Equal(Job{}, job)
	assert.Equal(Progress{}, progress)

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	// start drain job
	if count > 0 {
		done, job, err = d.Start(Job{Count: count, Rate: 100, Tick: time.Second, DrainFilter: df})
	} else {
		done, job, err = d.Start(Job{Rate: 100, Tick: time.Second, DrainFilter: df})
	}

	require.NoError(err)
	require.NotNil(done)

	assert.Equal(Job{Count: realCount, Rate: 100, Tick: time.Second, DrainFilter: df}, job)

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	{
		// test starting another drain job when there is one in progress
		done, job, err := d.Start(Job{Rate: 123, Tick: time.Minute})
		assert.Nil(done)
		assert.Error(err)
		assert.Equal(Job{}, job)
	}

	// get status of drain job in progress
	active, job, progress = d.Status()
	assert.True(active)
	assert.Equal(Job{Count: realCount, Rate: 100, Tick: time.Second, DrainFilter: df}, job)

	assert.Equal(Progress{Visited: 0, Drained: 0, Started: expectedStarted.UTC(), Finished: nil}, progress)

	go func() {
		ticks := realCount / 100
		if (realCount % 100) > 0 {
			ticks++
		}

		for i := 0; i < ticks; i++ {
			ticker <- time.Time{}
		}
	}()

	close(manager.pauseDisconnect)
	close(manager.pauseVisit)

	// make sure jobFinished is called and done channel is closed
	select {
	case <-done:
		// passed
	case <-time.After(5 * time.Second):
		assert.Fail("Drain failed to complete")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))

	if count > 0 && count <= totalCount-expectedSkipped {
		provider.Assert(t, "counter")(xmetricstest.Value(float64(count)))
	} else {
		provider.Assert(t, "counter")(xmetricstest.Value(float64(totalCount - expectedSkipped)))
	}

	// test cancel when not draining
	done, err = d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress = d.Status()
	assert.False(active)

	assert.Equal(Job{Count: realCount, Rate: 100, Tick: time.Second, DrainFilter: df}, job)

	if count > 0 && count <= (totalCount-expectedSkipped) {
		assert.Equal(count, progress.Visited)
		assert.Equal(count, progress.Drained)
		assert.Equal(totalCount-count, len(manager.devices))
	} else {
		assert.Equal(totalCount-expectedSkipped, progress.Visited)
		assert.Equal(totalCount-expectedSkipped, progress.Drained)
		assert.Equal(expectedSkipped, len(manager.devices))

	}

	assert.Equal(expectedStarted.UTC(), progress.Started)
	require.NotNil(progress.Finished)
	assert.Equal(expectedFinished.UTC(), *progress.Finished)

	assert.True(stopCalled)

}

func testDisconnectFilter(t *testing.T, deviceTypeOne deviceInfo, deviceTypeTwo deviceInfo, df DrainFilter, expectedSkipped int, count int) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil)
		logger   = logging.NewTestLogger(nil, t)

		// generate manager with devices that have two different metadatas
		manager = generateManagerWithDifferentDevices(assert, deviceTypeOne.claims, uint64(deviceTypeOne.count), deviceTypeTwo.claims, uint64(deviceTypeTwo.count))

		firstTime        = true
		expectedStarted  = time.Now()
		expectedFinished = expectedStarted.Add(10 * time.Minute)

		totalCount = deviceTypeOne.count + deviceTypeTwo.count

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

	defer d.Cancel() // cleanup in case of horribleness

	// test that cancel will error if there is not a drain job in progress
	done, err := d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	// test status when drain hasn't started
	active, job, progress := d.Status()
	assert.False(active)
	assert.Equal(Job{}, job)
	assert.Equal(Progress{}, progress)

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	// start drain job
	if count > 0 {
		done, job, err = d.Start(Job{Count: count, DrainFilter: df})
	} else {
		done, job, err = d.Start(Job{DrainFilter: df})
	}

	require.NoError(err)
	require.NotNil(done)

	if count > 0 {
		assert.Equal(Job{Count: count, DrainFilter: df}, job)
	} else {
		assert.Equal(Job{Count: totalCount, DrainFilter: df}, job)
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricDraining))
	provider.Assert(t, "counter")(xmetricstest.Value(0.0))

	{
		// test starting another drain job when there is one in progress
		done, job, err := d.Start(Job{Rate: 123, Tick: time.Minute})
		assert.Nil(done)
		assert.Error(err)
		assert.Equal(Job{}, job)
	}

	// get status of drain job in progress
	active, job, progress = d.Status()
	assert.True(active)
	if count > 0 {
		assert.Equal(Job{Count: count, DrainFilter: df}, job)
	} else {
		assert.Equal(Job{Count: totalCount, DrainFilter: df}, job)
	}

	assert.Equal(Progress{Visited: 0, Drained: 0, Started: expectedStarted.UTC(), Finished: nil}, progress)

	close(manager.pauseDisconnect)
	close(manager.pauseVisit)

	// make sure jobFinished is called and done channel is closed
	select {
	case <-done:
		// passed
	case <-time.After(5 * time.Second):
		assert.Fail("Drain failed to complete")
		return
	}

	provider.Assert(t, "state")(xmetricstest.Value(MetricNotDraining))

	if count > 0 && count <= totalCount-expectedSkipped {
		provider.Assert(t, "counter")(xmetricstest.Value(float64(count)))
	} else {
		provider.Assert(t, "counter")(xmetricstest.Value(float64(totalCount - expectedSkipped)))
	}

	// test cancel when not draining
	done, err = d.Cancel()
	assert.Nil(done)
	assert.Error(err)

	active, job, progress = d.Status()
	assert.False(active)

	if count > 0 {
		assert.Equal(Job{Count: count, DrainFilter: df}, job)
	} else {
		assert.Equal(Job{Count: totalCount, DrainFilter: df}, job)
	}

	if count > 0 && count <= (totalCount-expectedSkipped) {
		assert.Equal(count, progress.Visited)
		assert.Equal(count, progress.Drained)
		assert.Equal(totalCount-count, len(manager.devices))
	} else {
		assert.Equal(totalCount-expectedSkipped, progress.Visited)
		assert.Equal(totalCount-expectedSkipped, progress.Drained)
		assert.Equal(expectedSkipped, len(manager.devices))

	}

	assert.Equal(expectedStarted.UTC(), progress.Started)
	require.NotNil(progress.Finished)
	assert.Equal(expectedFinished.UTC(), *progress.Finished)
}

func TestDrainerWithFilter(t *testing.T) {
	var (
		filterKey   = "test"
		filterValue = "test1"
		df          = drainFilter{
			filter: &devicegate.FilterGate{
				FilterStore: devicegate.FilterStore(map[string]devicegate.Set{
					filterKey: devicegate.FilterSet(map[interface{}]bool{
						filterValue: true,
					}),
				}),
			},
			filterRequest: devicegate.FilterRequest{
				Key:    filterKey,
				Values: []interface{}{filterValue},
			},
		}

		metadata1 = map[string]interface{}{filterKey: "test"}
		metadata2 = map[string]interface{}{filterKey: filterValue}

		counts = [][]int{
			[]int{0, 0, 100},
			[]int{1, 0, 1},
			[]int{2, 0, 9},
			[]int{0, 1, 100},
			[]int{0, 2, 1},
			[]int{1, 1, 19},
			[]int{0, disconnectBatchSize - 1, 100},
			[]int{disconnectBatchSize - 1, 0, 20},
			[]int{0, disconnectBatchSize, 20},
			[]int{disconnectBatchSize, 0, 53},
			[]int{0, disconnectBatchSize + 1, 120},
			[]int{disconnectBatchSize + 1, 0, 400},
			[]int{89, 1709, 1091},
			[]int{1704, 43, 1000},
		}
	)

	for _, deviceCount := range counts {
		expectedSkip := deviceCount[0]
		devices := []deviceInfo{
			deviceInfo{count: deviceCount[0], claims: metadata1},
			deviceInfo{count: deviceCount[1], claims: metadata2},
		}

		t.Run(fmt.Sprintf("deviceCount=%d", deviceCount[0]+deviceCount[1]), func(t *testing.T) {
			t.Run("DrainAll", func(t *testing.T) {
				testDrainFilter(t, devices[0], devices[1], &df, expectedSkip, -1)
			})
			t.Run("DrainWithCount", func(t *testing.T) {
				testDrainFilter(t, devices[0], devices[1], &df, expectedSkip, deviceCount[2])
			})
			t.Run("DisconnectAll", func(t *testing.T) {
				testDisconnectFilter(t, devices[0], devices[1], &df, expectedSkip, -1)
			})
			t.Run("DisconnectWithCount", func(t *testing.T) {
				testDisconnectFilter(t, devices[0], devices[1], &df, expectedSkip, deviceCount[2])
			})
		})
	}
}

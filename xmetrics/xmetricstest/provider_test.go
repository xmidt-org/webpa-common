package xmetricstest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

func testNewProviderDefault(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewProvider(nil))
}

func testNewProviderGoodConfiguration(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewProvider(
		&xmetrics.Options{
			Metrics: []xmetrics.Metric{
				{Name: "Injected", Type: "counter"},
			},
		},
		func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "FromModule", Type: "gauge"},
			}
		},
	))
}

func testNewProviderBadConfiguration(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "duplicate", Type: "counter"},
				{Name: "duplicate", Type: "counter"},
			}
		})
	})
}

func testNewProviderUnsupportedType(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "unsupported", Type: "asdfasdfasdfasdf"},
			}
		})
	})
}

func TestNewProvider(t *testing.T) {
	t.Run("Default", testNewProviderDefault)
	t.Run("GoodConfiguration", testNewProviderGoodConfiguration)
	t.Run("BadConfiguration", testNewProviderBadConfiguration)
	t.Run("UnsupportedType", testNewProviderUnsupportedType)
}

func testProviderNewCounter(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		p = NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "preregistered", Type: "counter"},
				{Name: "gauge", Type: "gauge"},
			}
		})
	)

	require.NotNil(p)

	{
		preregistered := p.NewCounter("preregistered")
		require.NotNil(preregistered)
		assert.Implements((*xmetrics.Valuer)(nil), preregistered)
		assert.Implements((*Labeled)(nil), preregistered)
		assert.True(preregistered == p.NewCounter("preregistered"))
	}

	{
		adhoc := p.NewCounter("adhoc")
		require.NotNil(adhoc)
		assert.Implements((*xmetrics.Valuer)(nil), adhoc)
		assert.Implements((*Labeled)(nil), adhoc)
		assert.True(adhoc != p.NewCounter("preregistered"))
		assert.True(adhoc == p.NewCounter("adhoc"))
	}

	assert.Panics(func() {
		p.NewCounter("gauge")
	})
}

func testProviderNewGauge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		p = NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "preregistered", Type: "gauge"},
				{Name: "counter", Type: "counter"},
			}
		})
	)

	require.NotNil(p)

	{
		preregistered := p.NewGauge("preregistered")
		require.NotNil(preregistered)
		assert.Implements((*xmetrics.Valuer)(nil), preregistered)
		assert.Implements((*Labeled)(nil), preregistered)
		assert.True(preregistered == p.NewGauge("preregistered"))
	}

	{
		adhoc := p.NewGauge("adhoc")
		require.NotNil(adhoc)
		assert.Implements((*xmetrics.Valuer)(nil), adhoc)
		assert.Implements((*Labeled)(nil), adhoc)
		assert.True(adhoc != p.NewGauge("preregistered"))
		assert.True(adhoc == p.NewGauge("adhoc"))
	}

	assert.Panics(func() {
		p.NewGauge("counter")
	})
}

func testProviderNewHistogram(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		p = NewProvider(nil, func() []xmetrics.Metric {
			return []xmetrics.Metric{
				{Name: "preregistered", Type: "histogram"},
				{Name: "counter", Type: "counter"},
			}
		})
	)

	require.NotNil(p)

	{
		preregistered := p.NewHistogram("preregistered", 2)
		require.NotNil(preregistered)
		assert.Implements((*Labeled)(nil), preregistered)
		assert.True(preregistered == p.NewHistogram("preregistered", 4))
	}

	{
		adhoc := p.NewHistogram("adhoc", 5)
		require.NotNil(adhoc)
		assert.Implements((*Labeled)(nil), adhoc)
		assert.True(adhoc != p.NewHistogram("preregistered", 2))
		assert.True(adhoc == p.NewHistogram("adhoc", 6))
	}

	assert.Panics(func() {
		p.NewHistogram("counter", 3)
	})
}

func testProviderStop(t *testing.T) {
	var (
		assert = assert.New(t)
		p      = NewProvider(nil)
	)

	assert.NotPanics(p.Stop)
}

func testProviderExpect(t *testing.T) {
	t.Run("BadLabelsAndValues", func(t *testing.T) {
		var (
			assert = assert.New(t)
			p      = NewProvider(nil)
		)

		assert.Panics(func() {
			p.Expect("test", "one")
		})
	})
}

func testProviderAssert(t *testing.T) {
	t.Run("BadLabelsAndValues", func(t *testing.T) {
		var (
			assert   = assert.New(t)
			testingT = new(mockTestingT)

			p = NewProvider(nil)
		)

		assert.Panics(func() {
			p.Assert(testingT, "counter", "one")
		})

		testingT.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			firstCalled = false
			first       = func(testingT, string, interface{}) bool {
				firstCalled = true
				return true
			}

			lastCalled = false
			last       = func(testingT, string, interface{}) bool {
				lastCalled = true
				return true
			}

			testingT = new(mockTestingT)

			p = NewProvider(nil, func() []xmetrics.Metric {
				return []xmetrics.Metric{
					{Name: "preregistered_counter", Type: "counter"},
					{Name: "preregistered_gauge", Type: "gauge"},
					{Name: "preregistered_histogram", Type: "histogram"},
				}
			})
		)

		require.NotPanics(func() {
			p.NewCounter("preregistered_counter").Add(1.0)
			p.NewCounter("preregistered_counter").With("code", "500", "method", "POST").Add(2.0)
			p.NewGauge("preregistered_gauge").Set(15.0)
			p.NewCounter("adhoc_counter").Add(2.0)
		})

		assert.True(p.Assert(testingT, "preregistered_counter")(first, Counter, Value(1.0), last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		assert.True(p.Assert(testingT, "preregistered_counter", "method", "POST", "code", "500")(first, Counter, Value(2.0), last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		assert.True(p.Assert(testingT, "preregistered_gauge")(first, Gauge, last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		assert.True(p.Assert(testingT, "preregistered_gauge")(first, Value(15.0), last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		assert.True(p.Assert(testingT, "preregistered_histogram")(first, Histogram, last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		assert.True(p.Assert(testingT, "adhoc_counter")(first, Counter, Value(2.0), last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		testingT.AssertExpectations(t)
	})

	t.Run("Fail", func(t *testing.T) {
		var (
			assert = assert.New(t)

			firstCalled = false
			first       = func(testingT, string, interface{}) bool {
				firstCalled = true
				return true
			}

			lastCalled = false
			last       = func(testingT, string, interface{}) bool {
				lastCalled = true
				return true
			}

			nosuch = func(testingT, string, interface{}) bool {
				assert.Fail("nosuch should not have been called")
				return false
			}

			testingT = new(mockTestingT)

			p = NewProvider(nil, func() []xmetrics.Metric {
				return []xmetrics.Metric{
					{Name: "preregistered_counter", Type: "counter"},
				}
			})
		)

		testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Times(3)

		p.Assert(testingT, "nosuch")(nosuch)
		testingT.AssertNumberOfCalls(t, "Errorf", 1)

		assert.False(p.Assert(testingT, "preregistered_counter")(first, Gauge, Value(568.2), last))
		assert.True(firstCalled)
		assert.True(lastCalled)
		firstCalled = false
		lastCalled = false

		testingT.AssertExpectations(t)
	})
}

func testProviderAssertExpectations(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)

			firstCalled = false
			first       = func(testingT, string, interface{}) bool {
				firstCalled = true
				return true
			}

			lastCalled = false
			last       = func(testingT, string, interface{}) bool {
				lastCalled = true
				return true
			}

			testingT = new(mockTestingT)

			p = NewProvider(nil, func() []xmetrics.Metric {
				return []xmetrics.Metric{
					{Name: "preregistered_counter", Type: "counter"},
					{Name: "preregistered_gauge", Type: "gauge"},
					{Name: "preregistered_histogram", Type: "histogram"},
				}
			})
		)

		// no expect calls yet
		assert.True(p.AssertExpectations(t))

		p.Expect("preregistered_counter")(first).
			Expect("preregistered_counter")(Counter, Value(1.0)).
			Expect("preregistered_counter", "method", "POST", "code", "500")(Counter, Value(2.0)).
			Expect("preregistered_gauge")(Gauge).
			Expect("preregistered_gauge")(Value(15.0)).
			Expect("preregistered_histogram")(Histogram).
			Expect("adhoc_counter")(Counter, Value(2.0)).
			Expect("adhoc_counter")(last)

		require.NotPanics(func() {
			p.NewCounter("preregistered_counter").Add(1.0)
			p.NewCounter("preregistered_counter").With("code", "500", "method", "POST").Add(2.0)
			p.NewGauge("preregistered_gauge").Set(15.0)
			p.NewCounter("adhoc_counter").Add(2.0)
		})

		assert.True(p.AssertExpectations(testingT))
		assert.True(firstCalled)
		assert.True(lastCalled)
		testingT.AssertExpectations(t)
	})

	t.Run("Fail", func(t *testing.T) {
		var (
			assert = assert.New(t)

			firstCalled = false
			first       = func(testingT, string, interface{}) bool {
				firstCalled = true
				return true
			}

			lastCalled = false
			last       = func(testingT, string, interface{}) bool {
				lastCalled = true
				return true
			}

			nosuch = func(testingT, string, interface{}) bool {
				assert.Fail("nosuch should not have been called")
				return true
			}

			testingT = new(mockTestingT)

			p = NewProvider(nil, func() []xmetrics.Metric {
				return []xmetrics.Metric{
					{Name: "preregistered_counter", Type: "counter"},
				}
			})
		)

		p.Expect("preregistered_counter")(first).
			Expect("nosuch")(nosuch).
			Expect("preregistered_counter")(Gauge, Value(568.2)).
			Expect("preregistered_counter")(last)

		testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Times(3)
		assert.False(p.AssertExpectations(testingT))
		assert.True(firstCalled)
		assert.True(lastCalled)
		testingT.AssertExpectations(t)
	})
}

func TestProvider(t *testing.T) {
	t.Run("NewCounter", testProviderNewCounter)
	t.Run("NewGauge", testProviderNewGauge)
	t.Run("NewHistogram", testProviderNewHistogram)
	t.Run("Stop", testProviderStop)
	t.Run("Expect", testProviderExpect)
	t.Run("Assert", testProviderAssert)
	t.Run("AssertExpectations", testProviderAssertExpectations)
}

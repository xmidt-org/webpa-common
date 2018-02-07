package xmetricstest

import (
	"bytes"
	"testing"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testValueWrongMetricType(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		wrongType bytes.Buffer // just something that isn't a Valuer
	)

	testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Once()
	assert.False(
		Value(1.0)(testingT, "test", wrongType),
	)

	testingT.AssertExpectations(t)
}

func testValueFail(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		g = generic.NewGauge("test")
	)

	g.Set(1.0)
	testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Once()
	assert.False(
		Value(2.0)(testingT, "test", g),
	)

	testingT.AssertExpectations(t)
}

func testValueSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		g = generic.NewGauge("test")
	)

	g.Set(1.0)
	assert.True(
		Value(1.0)(testingT, "test", g),
	)

	testingT.AssertExpectations(t)
}

func TestValue(t *testing.T) {
	t.Run("WrongMetricType", testValueWrongMetricType)
	t.Run("Fail", testValueFail)
	t.Run("Success", testValueSuccess)
}

func testCounterFail(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		wrongType = NewGauge("test")
	)

	testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Once()
	assert.False(
		Counter(testingT, "test", wrongType),
	)

	testingT.AssertExpectations(t)
}

func testCounterSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		c = NewCounter("test")
	)

	assert.True(
		Counter(testingT, "test", c),
	)

	testingT.AssertExpectations(t)
}

func TestCounter(t *testing.T) {
	t.Run("Fail", testCounterFail)
	t.Run("Success", testCounterSuccess)
}

func testGaugeFail(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		wrongType = NewHistogram("test", 2)
	)

	testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Once()
	assert.False(
		Gauge(testingT, "test", wrongType),
	)

	testingT.AssertExpectations(t)
}

func testGaugeSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		g = NewGauge("test")
	)

	assert.True(
		Gauge(testingT, "test", g),
	)

	testingT.AssertExpectations(t)
}

func TestGauge(t *testing.T) {
	t.Run("Fail", testGaugeFail)
	t.Run("Success", testGaugeSuccess)
}

func testHistogramFail(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		wrongType = NewCounter("test")
	)

	testingT.On("Errorf", mock.MatchedBy(AnyMessage), mock.MatchedBy(AnyArguments)).Once()
	assert.False(
		Histogram(testingT, "test", wrongType),
	)

	testingT.AssertExpectations(t)
}

func testHistogramSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		testingT = new(mockTestingT)

		h = NewHistogram("test", 4)
	)

	assert.True(
		Histogram(testingT, "test", h),
	)

	testingT.AssertExpectations(t)
}

func TestHistogram(t *testing.T) {
	t.Run("Fail", testHistogramFail)
	t.Run("Success", testHistogramSuccess)
}

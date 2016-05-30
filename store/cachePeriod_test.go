package store

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCachePeriodString(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		actual   CachePeriod
		expected string
	}{
		{
			CachePeriod(-45),
			"never",
		},
		{
			CachePeriodNever,
			"never",
		},
		{
			CachePeriodForever,
			"forever",
		},
		{
			CachePeriod(1 * time.Hour),
			"1h0m0s",
		},
		{
			CachePeriod(23 * time.Minute),
			"23m0s",
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expected, record.actual.String())
	}
}

func TestCachePeriodNext(t *testing.T) {
	assert := assert.New(t)
	base := time.Now()

	var testData = []struct {
		actual   CachePeriod
		expected time.Time
	}{
		{
			CachePeriod(1 * time.Hour),
			base.Add(1 * time.Hour),
		},
		{
			CachePeriod(-15 * time.Minute),
			base.Add(-15 * time.Minute),
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expected, record.actual.Next(base))
	}
}

type containsCachePeriod struct {
	Period CachePeriod `json:"period"`
}

func TestCachePeriodMarshalJSON(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		container containsCachePeriod
		expected  string
	}{
		{
			containsCachePeriod{-1293871},
			`{"period": "never"}`,
		},
		{
			containsCachePeriod{CachePeriodNever},
			`{"period": "never"}`,
		},
		{
			containsCachePeriod{0},
			`{"period": "forever"}`,
		},
		{
			containsCachePeriod{CachePeriodForever},
			`{"period": "forever"}`,
		},
		{
			containsCachePeriod{CachePeriod(1 * time.Hour)},
			`{"period": "1h0m0s"}`,
		},
		{
			containsCachePeriod{CachePeriod(6*time.Hour + 23*time.Minute + 3*time.Second)},
			`{"period": "6h23m3s"}`,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		actual, err := json.Marshal(record.container)
		if assert.Nil(err) {
			assert.JSONEq(record.expected, string(actual))
		}
	}
}

func TestCachePeriodUnmarshalJSON(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		jsonValue string
		expected  containsCachePeriod
	}{
		{
			`{"period": "never"}`,
			containsCachePeriod{CachePeriodNever},
		},
		{
			`{"period": -1}`,
			containsCachePeriod{CachePeriodNever},
		},
		{
			`{"period": "-1"}`,
			containsCachePeriod{CachePeriodNever},
		},
		{
			`{"period": -348792}`,
			containsCachePeriod{CachePeriodNever},
		},
		{
			`{"period": 0}`,
			containsCachePeriod{CachePeriodForever},
		},
		{
			`{"period": "0"}`,
			containsCachePeriod{CachePeriodForever},
		},
		{
			`{"period": "forever"}`,
			containsCachePeriod{CachePeriodForever},
		},
		{
			`{"period": 34719}`,
			containsCachePeriod{CachePeriod(34719 * time.Second)},
		},
		{
			`{"period": "9034"}`,
			containsCachePeriod{CachePeriod(9034 * time.Second)},
		},
		{
			`{"period": "1h0m0s"}`,
			containsCachePeriod{CachePeriod(1 * time.Hour)},
		},
		{
			`{"period": "3h0m0s"}`,
			containsCachePeriod{CachePeriod(3 * time.Hour)},
		},
		{
			`{"period": "4h17m56s"}`,
			containsCachePeriod{CachePeriod(4*time.Hour + 17*time.Minute + 56*time.Second)},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		actual := containsCachePeriod{}
		if err := json.Unmarshal([]byte(record.jsonValue), &actual); assert.Nil(err) {
			assert.Equal(record.expected, actual)
		}
	}
}

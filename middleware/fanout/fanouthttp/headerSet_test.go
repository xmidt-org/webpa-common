package fanouthttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testHeaderSetAdd(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []HeaderSet{
			nil,
			HeaderSet{},
			NewHeaderSet("content-type"),
		}
	)

	for _, hs := range testData {
		stringValue := hs.String()
		for _, h := range hs {
			assert.Contains(stringValue, h)
		}

		originalLength := len(hs)
		hs.Add()
		assert.Equal(len(hs), originalLength)

		hs.Add("x-ANOTHER-header")
		assert.Contains(hs, "X-Another-Header")
		assert.Equal(len(hs), originalLength+1)

		stringValue = hs.String()
		assert.NotEmpty(stringValue)
		for _, h := range hs {
			assert.Contains(stringValue, h)
		}

		originalLength = len(hs)
		hs.Add("one-header", "two-header")
		assert.Contains(hs, "One-Header")
		assert.Contains(hs, "Two-Header")
		assert.Equal(len(hs), originalLength+2)

		stringValue = hs.String()
		assert.NotEmpty(stringValue)
		for _, h := range hs {
			assert.Contains(stringValue, h)
		}
	}
}

func testHeaderSetFilter(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			hs             HeaderSet
			actualTarget   http.Header
			actualSource   http.Header
			expectedTarget http.Header
		}{
			{
				nil,
				nil,
				http.Header{},
				nil,
			},
			{
				nil,
				http.Header{"Content-Type": []string{"shenanigans"}},
				http.Header{},
				http.Header{"Content-Type": []string{"shenanigans"}},
			},
			{
				NewHeaderSet("X-Test"),
				http.Header{"Content-Type": []string{"shenanigans"}},
				http.Header{"X-Test": []string{"value"}},
				http.Header{"Content-Type": []string{"shenanigans"}, "X-Test": []string{"value"}},
			},
			{
				NewHeaderSet("X-Test"),
				http.Header{"Content-Type": []string{"shenanigans"}},
				http.Header{},
				http.Header{"Content-Type": []string{"shenanigans"}},
			},
			{
				NewHeaderSet("X-Test"),
				nil,
				http.Header{"X-Test": []string{"value"}},
				http.Header{"X-Test": []string{"value"}},
			},
			{
				NewHeaderSet("X-Test"),
				http.Header{"X-Test": []string{"shenanigans"}},
				http.Header{"X-Test": []string{"value"}},
				http.Header{"X-Test": []string{"value"}},
			},
			{
				NewHeaderSet("X-Test"),
				http.Header{"Content-Type": []string{"shenanigans"}},
				http.Header{"X-Test": []string{"value1", "value2"}},
				http.Header{"Content-Type": []string{"shenanigans"}, "X-Test": []string{"value1", "value2"}},
			},
			{
				NewHeaderSet("X-Test"),
				nil,
				http.Header{"X-Test": []string{"value1", "value2"}},
				http.Header{"X-Test": []string{"value1", "value2"}},
			},
			{
				NewHeaderSet("X-Test"),
				http.Header{"X-Test": []string{"shenanigans"}},
				http.Header{"X-Test": []string{"value1", "value2"}},
				http.Header{"X-Test": []string{"value1", "value2"}},
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(
			record.expectedTarget,
			record.hs.Filter(record.actualTarget, record.actualSource),
		)
	}
}

func TestHeaderSet(t *testing.T) {
	t.Run("Add", testHeaderSetAdd)
	t.Run("Filter", testHeaderSetFilter)
}

func TestNewHeaderSet(t *testing.T) {
	assert := assert.New(t)

	{
		hs := NewHeaderSet()
		assert.Empty(hs)
	}

	{
		hs := NewHeaderSet([]string{}...)
		assert.Empty(hs)
	}

	{
		hs := NewHeaderSet("Content-Type")
		assert.Equal(HeaderSet{"Content-Type"}, hs)
	}

	{
		hs := NewHeaderSet("one-batch", "two-batch")
		assert.Equal(HeaderSet{"One-Batch", "Two-Batch"}, hs)
	}
}

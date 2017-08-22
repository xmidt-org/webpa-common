package httputil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleApplyHeadersUsingHttpHeader() {
	var (
		someHeaders = http.Header{
			"X-Application-Version": []string{"1.0.2"},
		}

		handler = ApplyHeaders(someHeaders)(
			http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				// this is production code
			}),
		)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	handler.ServeHTTP(response, request)
	fmt.Println(response.HeaderMap["X-Application-Version"][0])
	// Output:
	// 1.0.2
}

func ExampleApplyHeadersUsingSimpleMap() {
	var (
		someHeaders = map[string]string{
			"X-Application-Version": "1.0.2",
		}

		handler = ApplyHeaders(someHeaders)(
			http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				// this is production code
			}),
		)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	handler.ServeHTTP(response, request)
	fmt.Println(response.HeaderMap["X-Application-Version"][0])
	// Output:
	// 1.0.2
}

func testApplyHeadersUnsupported(t *testing.T, h interface{}) {
	assert.Panics(t, func() { ApplyHeaders(h) })
}

func testApplyHeaders(t *testing.T, h interface{}, expected http.Header) {
	const expectedOutput = "testApplyHeaders"

	var (
		assert = assert.New(t)

		delegate = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.Write([]byte(expectedOutput))
		})

		decorator = ApplyHeaders(h)(delegate)
		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
	)

	decorator.ServeHTTP(response, request)
	assert.Equal(expectedOutput, response.Body.String())

	for k, _ := range expected {
		assert.Equal(expected[k], response.HeaderMap[k])
	}
}

func TestApplyHeaders(t *testing.T) {
	t.Run("Unsupported", func(t *testing.T) {
		testApplyHeadersUnsupported(t, 123)
		testApplyHeadersUnsupported(t, []string{"foo", "bar"})
	})

	t.Run("Nil", func(t *testing.T) { testApplyHeaders(t, nil, nil) })

	t.Run("Empty", func(t *testing.T) {
		testApplyHeaders(t, http.Header{}, nil)
		testApplyHeaders(t, map[string][]string{}, nil)
		testApplyHeaders(t, map[string]string{}, nil)
	})

	testApplyHeaders(
		t,
		http.Header{
			"X-Something":      []string{"value"},
			"X-Something-Else": []string{"value1", "value2"},
		},
		http.Header{
			"X-Something":      []string{"value"},
			"X-Something-Else": []string{"value1", "value2"},
		},
	)

	testApplyHeaders(
		t,
		map[string][]string{
			"X-Something":      []string{"value"},
			"X-Something-Else": []string{"value1", "value2"},
		},
		http.Header{
			"X-Something":      []string{"value"},
			"X-Something-Else": []string{"value1", "value2"},
		},
	)

	testApplyHeaders(
		t,
		map[string]string{
			"X-Something":      "value1",
			"X-Something-Else": "value2",
		},
		http.Header{
			"X-Something":      []string{"value1"},
			"X-Something-Else": []string{"value2"},
		},
	)
}

package handler

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"mime"
	"net/http"
	"net/http/httptest"
)

func dummyHttpOperation() (response *httptest.ResponseRecorder, request *http.Request) {
	response = httptest.NewRecorder()
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		panic(err)
	}

	return
}

func assertValidJsonErrorResponse(assert *assert.Assertions, response *httptest.ResponseRecorder, expectedStatusCode int) {
	assert.Equal(
		expectedStatusCode,
		response.Code,
		"Expected status code %d, but got %d",
		expectedStatusCode,
		response.Code,
	)

	mediaType, _, err := mime.ParseMediaType(response.Header().Get(ContentTypeHeader))
	if assert.Nil(err) {
		assert.Equal(
			"application/json",
			mediaType,
			"Unexpected media type: %s",
			mediaType,
		)
	}

	message := make(map[string]interface{})
	assert.Nil(
		json.Unmarshal(response.Body.Bytes(), &message),
		"Invalid JSON error payload: %s",
		response.Body.String(),
	)
}

func assertJsonErrorResponse(assert *assert.Assertions, response *httptest.ResponseRecorder, expectedStatusCode int, expectedMessage string) {
	assert.Equal(
		expectedStatusCode,
		response.Code,
		"Expected status code %d, but got %d",
		expectedStatusCode,
		response.Code,
	)

	mediaType, _, err := mime.ParseMediaType(response.Header().Get(ContentTypeHeader))
	if assert.Nil(err) {
		assert.Equal(
			"application/json",
			mediaType,
			"Unexpected media type: %s",
			mediaType,
		)
	}

	assert.Equal(response.Header().Get(ContentTypeOptionsHeader), NoSniff)
	assert.JSONEq(fmt.Sprintf(`{"message": "%s"}`, expectedMessage), response.Body.String())
}

func assertContext(assert *assert.Assertions, expected map[interface{}]interface{}, actual context.Context) {
	for key, expectedValue := range expected {
		assert.Equal(expectedValue, actual.Value(key))
	}
}

type testContextHandler struct {
	assert          *assert.Assertions
	wasCalled       bool
	statusCode      int
	expectedContext map[interface{}]interface{}
}

func (handler *testContextHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	handler.wasCalled = true
	if handler.statusCode > 0 {
		response.WriteHeader(handler.statusCode)
	}

	assertContext(handler.assert, handler.expectedContext, ctx)
}

// panicContextHandler is a ContextHandler that always panics
type panicContextHandler struct {
	wasCalled bool
	value     interface{}
}

func (p *panicContextHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	p.wasCalled = true
	panic(p.value)
}

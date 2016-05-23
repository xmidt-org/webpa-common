package handler

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func assertJsonErrorResponse(t *testing.T, response *httptest.ResponseRecorder, expectedStatusCode int, expectedMessage string) {
	if expectedStatusCode != response.Code {
		t.Errorf("Expected status code %d, but got %d", expectedStatusCode, response.Code)
	}

	if mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type")); err != nil {
		t.Errorf("Unable to parse response Content-Type: %v", err)
	} else if !strings.EqualFold("application/json", mediaType) {
		t.Errorf("Unexpected media type: %s", mediaType)
	}

	assert.JSONEq(t, fmt.Sprintf(`{"message": "%s"}`, expectedMessage), response.Body.String())
}

// errorChainHandler is a ChainHandler that always writes an error to the response.
// It does not invoke the next handler.
type errorChainHandler struct {
	statusCode int
	message    string
}

func (e errorChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	WriteJsonError(response, e.statusCode, e.message)
}

// successChainHandler is a ChainHandler that always succeeds.  It invokes the next handler
// after first modifying the context with a key/value pair.
type successChainHandler struct {
	key   interface{}
	value interface{}
}

func (s successChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	next.ServeHTTP(
		context.WithValue(ctx, s.key, s.value),
		response,
		request,
	)
}

// panicHandler is a ChainHandler that always panics with the given value
type panicHandler struct {
	value interface{}
}

func (p panicHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	panic(p.value)
}

// invokeServeHttp is a helper function that invokes handler.ServeHttp() with
// dummy arguments.  The dummy arguments are returned for verification, if desired.
func invokeServeHttp(t *testing.T, handler http.Handler) (response *httptest.ResponseRecorder, request *http.Request) {
	response = httptest.NewRecorder()
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
		return
	}

	handler.ServeHTTP(response, request)
	t.Logf("response code: %d, response body: %s", response.Code, response.Body.String())
	return
}

type testContextHandler struct {
	t               *testing.T
	wasCalled       bool
	expectedContext map[interface{}]interface{}
}

func (handler *testContextHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	handler.t.Logf("ServeHTTP called: ctx=%v", ctx)
	handler.wasCalled = true
	for key, expected := range handler.expectedContext {
		actual := ctx.Value(key)
		if !reflect.DeepEqual(expected, actual) {
			handler.t.Errorf(
				"Expected context key [%v] to be [%v], but got [%v]",
				key,
				expected,
				actual,
			)
		}
	}
}

func (handler *testContextHandler) assertCalled(expected bool) {
	if expected && !handler.wasCalled {
		handler.t.Error("ContextHandler was not called")
	} else if !expected && handler.wasCalled {
		handler.t.Error("ContextHandler was called")
	}
}

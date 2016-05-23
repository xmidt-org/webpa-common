package handler

import (
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

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

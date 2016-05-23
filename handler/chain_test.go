package handler

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errorChainHandler struct {
	statusCode int
	message    string
}

func (e errorChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	WriteJsonError(response, e.statusCode, e.message)
}

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

type chainExpect struct {
	contextHandlerCalled bool
	statusCode           int
	message              string
}

func (expect *chainExpect) assert(assertions *assert.Assertions, response *httptest.ResponseRecorder, contextHandler *testContextHandler) {
	contextHandler.assertCalled(expect.contextHandlerCalled)
	assertions.Equal(expect.statusCode, response.Code)

	if len(expect.message) > 0 {
		mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
		assertions.NoError(err)
		mediaType = strings.ToLower(mediaType)
		assertions.Equal("application/json", mediaType)
		assertions.JSONEq(
			fmt.Sprintf(`{"message": "%s"}`, expect.message),
			response.Body.String(),
		)
	}
}

func TestDecorate(t *testing.T) {
	assertions := assert.New(t)

	var testData = []struct {
		chain          Chain
		contextHandler *testContextHandler
		expect         chainExpect
	}{
		{
			Chain{},
			&testContextHandler{
				t: t,
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				successChainHandler{
					123, "foobar",
				},
			},
			&testContextHandler{
				t:               t,
				expectedContext: map[interface{}]interface{}{123: "foobar"},
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				errorChainHandler{
					555, "an error message",
				},
			},
			&testContextHandler{
				t: t,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           555,
				message:              "an error message",
			},
		},
		{
			Chain{
				successChainHandler{
					123, "foobar",
				},
				successChainHandler{
					456, "asdf",
				},
				successChainHandler{
					"test", "giggity",
				},
			},
			&testContextHandler{
				t:               t,
				expectedContext: map[interface{}]interface{}{123: "foobar", 456: "asdf", "test": "giggity"},
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				successChainHandler{
					123, "foobar",
				},
				errorChainHandler{
					555, "an error message",
				},
			},
			&testContextHandler{
				t: t,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           555,
				message:              "an error message",
			},
		},
	}

	for _, record := range testData {
		ctx := context.Background()

		decorated := record.chain.Decorate(ctx, record.contextHandler)
		response, _ := invokeServeHttp(t, decorated)
		record.expect.assert(assertions, response, record.contextHandler)
	}
}

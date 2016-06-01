package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"testing"
)

type testChainHandler interface {
	setHandlers(chan<- testChainHandler)
}

// errorChainHandler is a ChainHandler that always writes an error to the response.
// It does not invoke the next handler.
type errorChainHandler struct {
	handlers   chan<- testChainHandler
	statusCode int
	message    string
}

func (e *errorChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	e.handlers <- e
	WriteJsonError(response, e.statusCode, e.message)
}

func (e *errorChainHandler) setHandlers(handlers chan<- testChainHandler) {
	e.handlers = handlers
}

// successChainHandler is a ChainHandler that always succeeds.  It invokes the next handler
// after first modifying the context with a key/value pair.
type successChainHandler struct {
	handlers chan<- testChainHandler
	key      interface{}
	value    interface{}
}

func (s *successChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	s.handlers <- s
	next.ServeHTTP(
		context.WithValue(ctx, s.key, s.value),
		response,
		request,
	)
}

func (s *successChainHandler) setHandlers(handlers chan<- testChainHandler) {
	s.handlers = handlers
}

// panicChainHandler is a ChainHandler that always panics
type panicChainHandler struct {
	handlers chan<- testChainHandler
	value    interface{}
}

func (p *panicChainHandler) ServeHTTP(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	p.handlers <- p
	panic(p.value)
}

func (p *panicChainHandler) setHandlers(handlers chan<- testChainHandler) {
	p.handlers = handlers
}

type chainExpect struct {
	contextHandlerCalled bool
	statusCode           int
	message              string
}

func TestDecorate(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		chain          Chain
		contextHandler *testContextHandler
		expect         chainExpect
	}{
		{
			Chain{},
			&testContextHandler{
				assert: assert,
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				&successChainHandler{
					key:   123,
					value: "foobar",
				},
			},
			&testContextHandler{
				assert:          assert,
				expectedContext: map[interface{}]interface{}{123: "foobar"},
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				&errorChainHandler{
					statusCode: 555,
					message:    "an error message",
				},
			},
			&testContextHandler{
				assert: assert,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           555,
				message:              "an error message",
			},
		},
		{
			Chain{
				&successChainHandler{
					key:   123,
					value: "foobar",
				},
				&successChainHandler{
					key:   456,
					value: "asdf",
				},
				&successChainHandler{
					key:   "test",
					value: "giggity",
				},
			},
			&testContextHandler{
				assert:          assert,
				expectedContext: map[interface{}]interface{}{123: "foobar", 456: "asdf", "test": "giggity"},
			},
			chainExpect{
				contextHandlerCalled: true,
				statusCode:           200,
			},
		},
		{
			Chain{
				&successChainHandler{
					key:   123,
					value: "foobar",
				},
				&errorChainHandler{
					statusCode: 555,
					message:    "an error message",
				},
			},
			&testContextHandler{
				assert: assert,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           555,
				message:              "an error message",
			},
		},
		{
			Chain{
				&panicChainHandler{value: "an error message"},
			},
			&testContextHandler{
				assert: assert,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           http.StatusInternalServerError,
				message:              "an error message",
			},
		},
		{
			Chain{
				&panicChainHandler{value: NewHttpError(598, "it's on fire!")},
			},
			&testContextHandler{
				assert: assert,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           598,
				message:              "it's on fire!",
			},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		ctx := context.Background()
		assert.Equal(len(record.chain), record.chain.Len())

		handlers := make(chan testChainHandler, record.chain.Len())
		for _, handler := range record.chain {
			handler.(testChainHandler).setHandlers(handlers)
		}

		decorated := record.chain.Decorate(ctx, record.contextHandler)
		response, request := dummyHttpOperation()
		decorated.ServeHTTP(response, request)
		assert.Equal(record.expect.contextHandlerCalled, record.contextHandler.wasCalled)

		if len(record.expect.message) > 0 {
			assertJsonErrorResponse(assert, response, record.expect.statusCode, record.expect.message)
		}

		// verify the order of handlers
		close(handlers)
		index := 0
		for calledHandler := range handlers {
			assert.Equal(
				record.chain[index],
				calledHandler,
				"Unexpected handler at %d",
				index,
			)

			index++
		}
	}
}

func ExampleChain() {
	logger := &logging.LoggerWriter{os.Stdout}
	ctx := fact.SetLogger(context.Background(), logger)

	contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		logger := fact.MustLogger(ctx)
		payloadJson, err := json.Marshal(fact.MustConvey(ctx))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal convey payload: %v\n", err)
			return
		}

		logger.Info("%s", payloadJson)
		logger.Info("%s", fact.MustDeviceId(ctx))
	})

	response, request := dummyHttpOperation()
	request.Header.Add(ConveyHeader, "eyJuYW1lIjoidmFsdWUifQ==")
	request.Header.Add(DeviceNameHeader, "mac:111122223333")

	Chain{
		Convey(),
		DeviceId(),
	}.Decorate(ctx, contextHandler).ServeHTTP(response, request)

	// Output:
	// [INFO]  {"name":"value"}
	// [INFO]  mac:111122223333
}

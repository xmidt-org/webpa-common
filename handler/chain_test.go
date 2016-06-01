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
				successChainHandler{
					123, "foobar",
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
				errorChainHandler{
					555, "an error message",
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
				successChainHandler{
					123, "foobar",
				},
				errorChainHandler{
					555, "an error message",
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
		ctx := context.Background()

		decorated := record.chain.Decorate(ctx, record.contextHandler)
		response, request := dummyHttpOperation()
		decorated.ServeHTTP(response, request)
		assert.Equal(record.expect.contextHandlerCalled, record.contextHandler.wasCalled)

		if len(record.expect.message) > 0 {
			assertJsonErrorResponse(assert, response, record.expect.statusCode, record.expect.message)
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

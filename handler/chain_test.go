package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type chainExpect struct {
	contextHandlerCalled bool
	statusCode           int
	message              string
}

func (expect *chainExpect) assert(t *testing.T, response *httptest.ResponseRecorder, contextHandler *testContextHandler) {
	contextHandler.assertCalled(expect.contextHandlerCalled)

	if len(expect.message) > 0 {
		assertJsonErrorResponse(t, response, expect.statusCode, expect.message)
	}
}

func TestDecorate(t *testing.T) {
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
		{
			Chain{
				panicHandler{
					"an error message",
				},
			},
			&testContextHandler{
				t: t,
			},
			chainExpect{
				contextHandlerCalled: false,
				statusCode:           http.StatusInternalServerError,
				message:              "an error message",
			},
		},
		{
			Chain{
				panicHandler{
					NewHttpError(598, "it's on fire!"),
				},
			},
			&testContextHandler{
				t: t,
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
		response, _ := invokeServeHttp(t, decorated)
		record.expect.assert(t, response, record.contextHandler)
	}
}

func ExampleTypicalChain() {
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

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create request: %v\n", err)
		return
	}

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

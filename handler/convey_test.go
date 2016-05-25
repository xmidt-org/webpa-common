package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"testing"
)

func ExampleConvey() {
	contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		payload := fact.MustConvey(ctx)
		payloadJson, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not marshal payload: %v\n", err)
		} else {
			fmt.Printf("%s\n", payloadJson)
		}
	})

	response, request := dummyHttpOperation()
	request.Header.Add(ConveyHeader, "eyJuYW1lIjoidmFsdWUifQ==")
	Convey().ServeHTTP(context.Background(), response, request, contextHandler)

	// Output: {"name":"value"}
}

func TestConveyCustom(t *testing.T) {
	assert := assert.New(t)
	const expectedPayloadJson string = `{"name": "value"}`

	for _, headerName := range []string{ConveyHeader, "X-Some-Header"} {
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
			contextHandlerCalled := false
			contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
				contextHandlerCalled = true
				payload := fact.MustConvey(ctx)
				actualPayloadJson, err := json.Marshal(payload)
				if assert.Nil(err) {
					assert.JSONEq(expectedPayloadJson, string(actualPayloadJson))
				}
			})

			var encodedPayload bytes.Buffer
			encoder := base64.NewEncoder(encoding, &encodedPayload)
			if _, err := encoder.Write([]byte(expectedPayloadJson)); assert.Nil(err) {
				assert.Nil(encoder.Close())
			}

			response, request := dummyHttpOperation()
			request.Header.Add(headerName, encodedPayload.String())
			ConveyCustom(headerName, encoding).ServeHTTP(
				context.Background(),
				response,
				request,
				contextHandler,
			)

			assert.Equal(200, response.Code)
			assert.True(contextHandlerCalled)
		}
	}
}

// BUG: https://www.teamccp.com/jira/browse/WEBPA-787
func TestConveyNotAvailable(t *testing.T) {
	assert := assert.New(t)

	contextHandlerCalled := false
	contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		contextHandlerCalled = true
		payload, ok := fact.Convey(ctx)
		assert.Nil(payload)
		assert.False(ok)
	})

	response, request := dummyHttpOperation()
	request.Header.Add(ConveyHeader, "not-available")
	Convey().ServeHTTP(
		context.Background(),
		response,
		request,
		contextHandler,
	)

	assert.True(contextHandlerCalled)
}

func TestConveyInvalid(t *testing.T) {
	assert := assert.New(t)

	contextHandlerCalled := false
	contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		contextHandlerCalled = true
		payload, ok := fact.Convey(ctx)
		assert.Nil(payload)
		assert.False(ok)
	})

	response, request := dummyHttpOperation()
	request.Header.Add(ConveyHeader, "this is an invalid convey value")
	Convey().ServeHTTP(
		context.Background(),
		response,
		request,
		contextHandler,
	)

	assert.False(contextHandlerCalled)
	assertValidJsonErrorResponse(assert, response, http.StatusBadRequest)
}

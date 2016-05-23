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
	"net/http/httptest"
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

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create request: %v\n", err)
		return
	}

	request.Header.Add(ConveyHeader, "eyJuYW1lIjoidmFsdWUifQ==")

	Chain{
		Convey(),
	}.Decorate(context.Background(), contextHandler).ServeHTTP(response, request)
	// Output: {"name":"value"}
}

func TestConveyCustom(t *testing.T) {
	assertions := assert.New(t)
	const expectedPayloadJson string = `{"name": "value"}`

	for _, headerName := range []string{ConveyHeader, "X-Some-Header"} {
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
			contextHandlerCalled := false

			contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
				contextHandlerCalled = true
				payload := fact.MustConvey(ctx)
				actualPayloadJson, err := json.Marshal(payload)
				if err != nil {
					t.Errorf("Could not marshal payload: %v", err)
				} else {
					assertions.JSONEq(expectedPayloadJson, string(actualPayloadJson))
				}
			})

			var encodedPayload bytes.Buffer
			encoder := base64.NewEncoder(encoding, &encodedPayload)
			if _, err := encoder.Write([]byte(expectedPayloadJson)); err != nil {
				t.Fatalf("Unable to write encoded JSON: %v", err)
			} else if err = encoder.Close(); err != nil {
				t.Fatalf("Unable to close encoder: %v", err)
			}

			response := httptest.NewRecorder()
			request, err := http.NewRequest("GET", "", nil)
			if err != nil {
				t.Fatalf("Unable to create request: %v", err)
			}

			request.Header.Add(headerName, encodedPayload.String())
			ConveyCustom(headerName, encoding).ServeHTTP(
				context.Background(),
				response,
				request,
				contextHandler,
			)

			if response.Code != 200 {
				t.Errorf("Invalid response code %d", response.Code)
			}

			if !contextHandlerCalled {
				t.Fatal("Context handler was not called")
			}
		}
	}
}

package fanouthttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/middleware/fanout"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func testVariablesToHeadersValid(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			first, second  string
			rest           []string
			urlPattern     string
			url            string
			expectedHeader http.Header
		}{
			{
				"deviceID", "X-Webpa-Device-Name",
				nil,
				"/novariables",
				"/novariables",
				http.Header{},
			},
			{
				"deviceID", "X-Webpa-Device-Name",
				nil,
				"/device/{deviceID}/stat",
				"/device/mac:1234/stat",
				http.Header{"X-Webpa-Device-Name": []string{"mac:1234"}},
			},
			{
				"deviceID", "X-Webpa-Device-Name",
				nil,
				"/device/{nosuch}/stat",
				"/device/mac:1234/stat",
				http.Header{},
			},
			{
				"deviceID", "x-webpa-DEVICE-Name",
				nil,
				"/device/{deviceID}/stat",
				"/device/mac:1234/stat",
				http.Header{"X-Webpa-Device-Name": []string{"mac:1234"}},
			},
			{
				"deviceID", "X-Webpa-Device-Name",
				[]string{"other", "X-Other"},
				"/novariables",
				"/novariables",
				http.Header{},
			},
			{
				"deviceID", "X-Webpa-Device-Name",
				[]string{"other", "X-Other"},
				"/device/{deviceID}/{other}/stat",
				"/device/mac:1234/asdf/stat",
				http.Header{"X-Webpa-Device-Name": []string{"mac:1234"}, "X-Other": []string{"asdf"}},
			},
			{
				"deviceID", "X-Webpa-Device-Name",
				[]string{"other", "X-Other"},
				"/device/{nosuch}/{other}/stat",
				"/device/mac:1234/asdf/stat",
				http.Header{"X-Other": []string{"asdf"}},
			},
			{
				"deviceID", "x-webpa-DEVICE-Name",
				[]string{"other", "X-oTHER"},
				"/device/{deviceID}/{other}/stat",
				"/device/mac:1234/asdf/stat",
				http.Header{"X-Webpa-Device-Name": []string{"mac:1234"}, "X-Other": []string{"asdf"}},
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			v2h           = VariablesToHeaders(record.first, record.second, record.rest...)
			handlerCalled = false
			handler       = func(response http.ResponseWriter, original *http.Request) {
				// pretend this is handling a fanout request ....

				var (
					ctx       = fanout.NewContext(context.Background(), &fanoutRequest{original: original})
					component = httptest.NewRequest("GET", "/", nil)
				)

				assert.Equal(ctx, v2h(ctx, component))
				assert.Equal(record.expectedHeader, component.Header)
				handlerCalled = true
			}

			router   = mux.NewRouter()
			request  = httptest.NewRequest("GET", record.url, nil)
			response = httptest.NewRecorder()
		)

		router.HandleFunc(record.urlPattern, handler)
		router.ServeHTTP(response, request)
		assert.True(handlerCalled)
	}
}

func testVariablesToHeadersPanic(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		VariablesToHeaders("first", "second", "oh no!")
	})
}

func TestVariablesToHeaders(t *testing.T) {
	t.Run("Valid", testVariablesToHeadersValid)
	t.Run("Panics", testVariablesToHeadersPanic)
}

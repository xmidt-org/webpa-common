package xresolver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: NewResolver(DefaultDialer, logging.NewTestLogger(nil, t)).DialContext,
		},
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	assert.NoError(err)
	assert.Equal(200, res.StatusCode)
}

/****************** BEGIN MOCK DECLARATIONS ***********************/
type mockLookUp struct {
	mock.Mock
}

func (m *mockLookUp) LookupRoutes(ctx context.Context, host string) ([]Route, error) {
	args := m.Called(ctx, host)
	return args.Get(0).([]Route), args.Error(1)
}

/******************* END MOCK DECLARATIONS ************************/

func TestClientWithResolver(t *testing.T) {
	assert := assert.New(t)

	customhost := "custom.host.com"
	customport := "8080"
	expectedBody := "Hello World\n"

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expectedBody)
	}))
	defer serverA.Close()

	route, err := CreateRoute(serverA.URL)
	assert.NoError(err)

	fakeLookUp := new(mockLookUp)
	fakeLookUp.On("LookupRoutes", mock.Anything, customhost).Return([]Route{route}, nil)
	r := NewResolver(DefaultDialer, logging.NewTestLogger(nil, t), fakeLookUp)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:       r.DialContext,
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequest("GET", "http://"+customhost+":"+customport, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	if assert.NoError(err) {
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal(expectedBody, string(body))
		fakeLookUp.AssertExpectations(t)
	}

	// Remove CustomLook up
	err = r.Remove(fakeLookUp)
	assert.NoError(err)

	req, err = http.NewRequest("GET", "http://"+customhost+":"+customport, nil)
	assert.NoError(err)

	res, err = client.Do(req)
	assert.Error(err)
}

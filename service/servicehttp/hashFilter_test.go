package servicehttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testNewHashFilterNoSelf(t *testing.T) {
	testData := [][]string{
		nil,
		[]string{},
		[]string{""},
		[]string{" "},
		[]string{"  ", ""},
		[]string{"\t\r", "", "   "},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				accessor = new(service.MockAccessor)
				reject   = errors.New("reject")

				f = NewHashFilter(accessor, reject, record...)
			)

			require.NotNil(f)
			assert.NoError(f.Allow(new(http.Request)))
			accessor.AssertExpectations(t)
		})
	}
}

func testNewHashFilterPass(t *testing.T) {
	for _, selfCount := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(selfCount), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				accessor = new(service.MockAccessor)
				ids      = make([]device.ID, selfCount)
				self     = make([]string, selfCount)
			)

			for i := 0; i < selfCount; i++ {
				ids[i] = device.IntToMAC(uint64(i))
				self[i] = fmt.Sprintf("instance-%d", i)
				accessor.On("Get", ids[i].Bytes()).Return(self[i], error(nil)).Times(2)
			}

			f := NewHashFilter(accessor, errors.New("reject"), self...)
			require.NotNil(f)

			for i := 0; i < selfCount; i++ {
				assert.NoError(
					f.Allow(
						device.WithIDRequest(ids[i], httptest.NewRequest("GET", "/", nil)),
					),
				)

				request := httptest.NewRequest("GET", "/", nil)
				request.Header.Set(device.DeviceNameHeader, string(ids[i]))
				assert.NoError(f.Allow(request))
			}

			accessor.AssertExpectations(t)
		})
	}
}

func testNewHashFilterReject(t *testing.T) {
	for _, selfCount := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(selfCount), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				logger   = logging.NewTestLogger(nil, t)
				reject   = errors.New("reject")
				accessor = new(service.MockAccessor)
				self     = make([]string, selfCount)
			)

			accessor.On("Get", mock.MatchedBy(func([]byte) bool { return true })).Return("notSelf", error(nil)).Times(2)

			for i := 0; i < selfCount; i++ {
				self[i] = fmt.Sprintf("instance-%d", i)
			}

			f := NewHashFilter(accessor, reject, self...)
			require.NotNil(f)

			assert.Equal(
				reject,
				f.Allow(
					device.WithIDRequest(
						device.ID("mac:112233445566"),
						httptest.NewRequest("GET", "/", nil).WithContext(logging.WithLogger(context.Background(), logger)),
					),
				),
			)

			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set(device.DeviceNameHeader, "mac:112233445566")
			request = request.WithContext(logging.WithLogger(request.Context(), logger))
			assert.Equal(reject, f.Allow(request))

			accessor.AssertExpectations(t)
		})
	}
}

func testNewHashFilterParseError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id       = "asdlfkj039w485;lkjsd,.fjaw94385"
		accessor = new(service.MockAccessor)
	)

	f := NewHashFilter(accessor, errors.New("reject"), "instance-1")
	require.NotNil(f)

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.Error(f.Allow(request))

	accessor.AssertExpectations(t)
}

func testNewHashFilterHashError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id       = device.IntToMAC(999)
		hashErr  = errors.New("hash")
		accessor = new(service.MockAccessor)
	)

	accessor.On("Get", mock.MatchedBy(func([]byte) bool { return true })).Return("", hashErr).Times(2)

	f := NewHashFilter(accessor, errors.New("reject"), "instance-1")
	require.NotNil(f)

	assert.Equal(
		hashErr,
		f.Allow(
			device.WithIDRequest(id, httptest.NewRequest("GET", "/", nil)),
		),
	)

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.Equal(hashErr, f.Allow(request))

	accessor.AssertExpectations(t)
}

func TestNewHashFilter(t *testing.T) {
	t.Run("NoSelf", testNewHashFilterNoSelf)
	t.Run("Pass", testNewHashFilterPass)
	t.Run("Reject", testNewHashFilterReject)
	t.Run("ParseError", testNewHashFilterParseError)
	t.Run("HashError", testNewHashFilterHashError)
}

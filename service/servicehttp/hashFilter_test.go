package servicehttp

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/device"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/service"
)

func testNewHashFilterNoAccessor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		self = func(string) bool {
			assert.Fail("The self predicate should not have been called")
			return false
		}

		hf = NewHashFilter(nil, errors.New("reject"), self)
	)

	require.NotNil(hf)
	assert.NoError(hf.Allow(httptest.NewRequest("GET", "/", nil)))
}

func testNewHashFilterNoReject(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a = new(service.MockAccessor)

		self = func(string) bool {
			assert.Fail("The self predicate should not have been called")
			return false
		}

		hf = NewHashFilter(a, nil, self)
	)

	require.NotNil(hf)
	assert.NoError(hf.Allow(httptest.NewRequest("GET", "/", nil)))
	a.AssertExpectations(t)
}

func testNewHashFilterNoSelf(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		a  = new(service.MockAccessor)
		hf = NewHashFilter(a, errors.New("reject"), nil)
	)

	require.NotNil(hf)
	assert.NoError(hf.Allow(httptest.NewRequest("GET", "/", nil)))
	a.AssertExpectations(t)
}

func testNewHashFilterParseError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id   = "asdlfkj039w485;lkjsd,.fjaw94385"
		self = func(string) bool {
			assert.Fail("The self predicate should not have been called")
			return false
		}

		accessor = new(service.MockAccessor)
		hf       = NewHashFilter(accessor, errors.New("reject"), self)
	)

	require.NotNil(hf)

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.Error(hf.Allow(request))

	accessor.AssertExpectations(t)
}

func testNewHashFilterHashError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id   = device.IntToMAC(999)
		self = func(string) bool {
			assert.Fail("The self predicate should not have been called")
			return false
		}

		hashErr  = errors.New("hash")
		accessor = new(service.MockAccessor)
		hf       = NewHashFilter(accessor, errors.New("reject"), self)
	)

	require.NotNil(hf)
	accessor.On("Get", mock.MatchedBy(func([]byte) bool { return true })).Return("", hashErr).Times(2)

	assert.Equal(
		hashErr,
		hf.Allow(
			device.WithIDRequest(id, httptest.NewRequest("GET", "/", nil)),
		),
	)

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.Equal(hashErr, hf.Allow(request))

	accessor.AssertExpectations(t)
}

func testNewHashFilterAllow(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id               = device.IntToMAC(47283)
		expectedInstance = "instance"
		ctx              = logging.WithLogger(context.Background(), logging.NewTestLogger(nil, t))

		selfCalled = false
		self       = func(actualInstance string) bool {
			selfCalled = true
			assert.Equal(expectedInstance, actualInstance)
			return true
		}

		a  = new(service.MockAccessor)
		hf = NewHashFilter(a, errors.New("reject"), self)
	)

	require.NotNil(hf)
	a.On("Get", mock.MatchedBy(func(k []byte) bool { return string(id) == string(k) })).Return(expectedInstance, error(nil)).Once()

	request := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.NoError(hf.Allow(request))

	assert.True(selfCalled)
	a.AssertExpectations(t)
}

func testNewHashFilterReject(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		id               = device.IntToMAC(93723)
		expectedInstance = "instance"
		ctx              = logging.WithLogger(context.Background(), logging.NewTestLogger(nil, t))

		selfCalled = false
		self       = func(actualInstance string) bool {
			selfCalled = true
			assert.Equal(expectedInstance, actualInstance)
			return false
		}

		a           = new(service.MockAccessor)
		expectedErr = errors.New("expected")
		hf          = NewHashFilter(a, expectedErr, self)
	)

	require.NotNil(hf)
	a.On("Get", mock.MatchedBy(func(k []byte) bool { return string(id) == string(k) })).Return(expectedInstance, error(nil)).Once()

	request := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	request.Header.Set(device.DeviceNameHeader, string(id))
	assert.Equal(expectedErr, hf.Allow(request))

	assert.True(selfCalled)
	a.AssertExpectations(t)
}

func TestNewHashFilter(t *testing.T) {
	t.Run("NoAccessor", testNewHashFilterNoAccessor)
	t.Run("NoReject", testNewHashFilterNoReject)
	t.Run("NoSelf", testNewHashFilterNoSelf)
	t.Run("ParseError", testNewHashFilterParseError)
	t.Run("HashError", testNewHashFilterHashError)
	t.Run("Allow", testNewHashFilterAllow)
	t.Run("Reject", testNewHashFilterReject)
}

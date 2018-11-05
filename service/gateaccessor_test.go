package service

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

/****************** BEGIN MOCK DECLARATIONS ***********************/

type mockGate struct {
	mock.Mock
}

func (r mockGate) Raise() bool {
	args := r.Called()
	return args.Bool(0)
}

func (r mockGate) Lower() bool {
	args := r.Called()
	return args.Bool(0)
}

func (r mockGate) Open() bool {
	args := r.Called()
	return args.Bool(0)
}

func (r mockGate) State() (bool, time.Time) {
	args := r.Called()
	return args.Bool(0), args.Get(1).(time.Time)
}

func (r mockGate) String() string {
	args := r.Called()
	return args.String(0)
}

/******************* END MOCK DECLARATIONS ************************/

func TestGateAccessor(t *testing.T) {
	assert := assert.New(t)

	accessor := new(MockAccessor)
	gate := new(mockGate)

	gateAcessor := GateAccessor(gate, accessor)

	instance := "a valid instance"

	gate.On("Open").Return(false).Once()
	accessor.On("Get", []byte("testA")).Return(instance, nil)
	i, err := gateAcessor.Get([]byte("testA"))
	assert.Equal(instance, i)
	assert.Equal(errGateClosed, err)

	gate.On("Open").Return(true).Once()
	accessor.On("Get", []byte("testB")).Return(instance, nil)
	i, err = gateAcessor.Get([]byte("testB"))
	assert.Equal(instance, i)
	assert.NoError(err)

	expectedErr := errors.New("no instances")
	accessor.On("Get", []byte("testC")).Return(instance, expectedErr)
	i, err = gateAcessor.Get([]byte("testC"))
	assert.Equal(instance, i)
	assert.Equal(expectedErr, err)

	defaultGateAccessor := GateAccessor(nil, nil)
	i, err = defaultGateAccessor.Get([]byte("testC"))
	assert.Empty(i)
	assert.Error(err)

	gate.AssertExpectations(t)
	accessor.AssertExpectations(t)
}

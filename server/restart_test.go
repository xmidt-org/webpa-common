package server

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
	"time"
)

func TestEmptyServer(t *testing.T) {
	assert := assert.New(t)

	err := BeginRestartableServer(nil, nil, make(chan struct{}, 1), nil)
	assert.Error(err)

	err = BeginRestartableServer(func() error {
		return nil
	}, nil, make(chan struct{}, 1), nil)
	assert.Error(err)

	err = BeginRestartableServer(func() error {
		return nil
	}, func(ctx context.Context) error {
		return nil
	}, make(chan struct{}, 1), nil)
	assert.NoError(err)
}

func TestRestartServerByErr(t *testing.T) {
	assert := assert.New(t)

	mockServer := new(mockServerable)
	mockServer.On("Serve").Return(errors.New("unknown error")).Once()
	mockServer.On("Serve").Return(http.ErrServerClosed).Once()

	err := BeginRestartableServer(mockServer.Serve, mockServer.Shutdown, make(chan struct{}, 1), nil)
	assert.NoError(err)
	time.Sleep(time.Second)
	mockServer.AssertExpectations(t)
}

func TestRestartServerByChan(t *testing.T) {
	assert := assert.New(t)

	mockServer := new(mockServerable)
	mockServer.On("Serve").Return(errors.New("unknown error"))
	mockServer.On("Shutdown", mock.Anything).Return(nil).Once()

	done := make(chan struct{}, 1)

	err := BeginRestartableServer(mockServer.Serve, mockServer.Shutdown, done, nil)

	assert.NoError(err)
	time.Sleep(time.Millisecond)
	done <- struct{}{}
	time.Sleep(time.Second)
	mockServer.AssertExpectations(t)
}

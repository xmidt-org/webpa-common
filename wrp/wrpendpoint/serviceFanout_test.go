package wrpendpoint

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testNewServiceFanoutNoConfiguredServices(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
		request = WrapAsRequest(logging.NewTestLogger(nil, t), new(wrp.Message))
	)

	for _, empty := range []map[string]Service{nil, map[string]Service{}} {
		fanout := NewServiceFanout(empty)
		require.NotNil(fanout)

		response, err := fanout.ServeWRP(context.Background(), request)
		assert.Nil(response)
		assert.Error(err)
	}
}

func testNewServiceFanoutSuccessFirst(t *testing.T, serviceCount int) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithCancel(context.Background())
		expectedRequest     = WrapAsRequest(logging.DefaultCaller(logging.NewTestLogger(nil, t)), new(wrp.Message))
		expectedResponse    = WrapAsResponse(new(wrp.Message))

		services     = make(map[string]Service, serviceCount)
		failuresDone = new(sync.WaitGroup)
		failureBlock = make(chan time.Time)
	)

	failuresDone.Add(serviceCount - 1)
	for i := 0; i < serviceCount; i++ {
		service := new(mockService)
		if i == 0 {
			services["success"] = service
			service.On("ServeWRP", expectedCtx, expectedRequest).
				Return(expectedResponse, error(nil)).Once()
		} else {
			services[fmt.Sprintf("failure#%d", i)] = service
			service.On("ServeWRP", expectedCtx, expectedRequest).
				Return(nil, errors.New(fmt.Sprintf("error#%d", i))).WaitUntil(failureBlock).Run(func(mock.Arguments) { failuresDone.Done() }).Once()
		}
	}

	fanout := NewServiceFanout(services)
	require.NotNil(fanout)

	response, err := fanout.ServeWRP(expectedCtx, expectedRequest)
	assert.NoError(err)
	require.NotNil(response)

	require.Equal(1, len(response.Spans()))
	assert.Equal("success", response.Spans()[0].Name())
	assert.NoError(response.Spans()[0].Error())

	// let the services that will fail go ahead an run
	close(failureBlock)
	failuresDone.Wait()
	cancel()
	for _, s := range services {
		s.(*mockService).AssertExpectations(t)
	}
}

func testNewServiceFanoutSuccessLast(t *testing.T, serviceCount int) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithCancel(context.Background())
		expectedRequest     = WrapAsRequest(logging.DefaultCaller(logging.NewTestLogger(nil, t)), new(wrp.Message))
		expectedResponse    = WrapAsResponse(new(wrp.Message))

		services     = make(map[string]Service, serviceCount)
		failuresDone = new(sync.WaitGroup)
		successBlock = make(chan time.Time)
	)

	failuresDone.Add(serviceCount - 1)
	for i := 0; i < serviceCount; i++ {
		service := new(mockService)
		if i == 0 {
			services["success"] = service
			service.On("ServeWRP", expectedCtx, expectedRequest).
				Return(expectedResponse, error(nil)).WaitUntil(successBlock).Once()
		} else {
			services[fmt.Sprintf("failure#%d", i)] = service
			service.On("ServeWRP", expectedCtx, expectedRequest).
				Return(nil, errors.New(fmt.Sprintf("error#%d", i))).Run(func(mock.Arguments) { failuresDone.Done() }).Once()
		}
	}

	fanout := NewServiceFanout(services)
	require.NotNil(fanout)

	go func() {
		// only once the failures are done do we allow the success to execute
		failuresDone.Wait()
		close(successBlock)
	}()

	response, err := fanout.ServeWRP(expectedCtx, expectedRequest)
	assert.NoError(err)
	require.NotNil(response)

	// we can't be exact here, since race detection and coverage can play havoc
	// with the timing of selects
	require.True(len(response.Spans()) >= (serviceCount - 1))
	successFound := false
	for _, s := range response.Spans() {
		if s.Name() == "success" {
			assert.NoError(s.Error())
			successFound = true
		} else {
			assert.Error(s.Error())
		}
	}

	assert.True(successFound)

	cancel()
	for _, s := range services {
		s.(*mockService).AssertExpectations(t)
	}
}

func testNewServiceFanoutTimeout(t *testing.T, serviceCount int) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
		expectedRequest     = WrapAsRequest(logging.DefaultCaller(logging.NewTestLogger(nil, t)), new(wrp.Message))

		services         = make(map[string]Service, serviceCount)
		allServicesDone  = new(sync.WaitGroup)
		allServicesBlock = make(chan time.Time)
	)

	defer cancel()
	allServicesDone.Add(serviceCount)
	for i := 0; i < serviceCount; i++ {
		service := new(mockService)
		services[fmt.Sprintf("timesout#%d", i)] = service
		service.On("ServeWRP", expectedCtx, expectedRequest).
			Return(nil, errors.New("does not matter as this is dropped")).
			WaitUntil(allServicesBlock).Run(func(mock.Arguments) { allServicesDone.Done() }).Once()
	}

	fanout := NewServiceFanout(services)
	require.NotNil(fanout)

	response, err := fanout.ServeWRP(expectedCtx, expectedRequest)
	assert.Nil(response)

	spanError, ok := err.(tracing.SpanError)
	require.True(ok)
	assert.Empty(spanError.Spans())
	assert.Equal(context.DeadlineExceeded, spanError.Err())

	close(allServicesBlock)
	allServicesDone.Wait()
	for _, s := range services {
		s.(*mockService).AssertExpectations(t)
	}
}

func testNewServiceFanoutAllFail(t *testing.T, serviceCount int) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithCancel(context.Background())
		expectedRequest     = WrapAsRequest(logging.DefaultCaller(logging.NewTestLogger(nil, t)), new(wrp.Message))

		services = make(map[string]Service, serviceCount)
	)

	defer cancel()
	for i := 0; i < serviceCount; i++ {
		service := new(mockService)
		services[fmt.Sprintf("failure#%d", i)] = service
		service.On("ServeWRP", expectedCtx, expectedRequest).
			Return(nil, errors.New(fmt.Sprintf("error#%d", i))).Once()
	}

	fanout := NewServiceFanout(services)
	require.NotNil(fanout)

	response, err := fanout.ServeWRP(expectedCtx, expectedRequest)
	assert.Nil(response)

	spanError, ok := err.(tracing.SpanError)
	require.True(ok)
	require.Equal(serviceCount, len(spanError.Spans()))
	assert.Equal(spanError.Err(), spanError.Spans()[serviceCount-1].Error())

	for _, s := range services {
		s.(*mockService).AssertExpectations(t)
	}
}

func TestNewServiceFanout(t *testing.T) {
	t.Run("NoConfiguredServices", testNewServiceFanoutNoConfiguredServices)

	t.Run("SuccessFirst", func(t *testing.T) {
		for c := 1; c <= 5; c++ {
			t.Run(fmt.Sprintf("ServiceCount=%d", c), func(t *testing.T) {
				testNewServiceFanoutSuccessFirst(t, c)
			})
		}
	})

	t.Run("SuccessLast", func(t *testing.T) {
		for c := 1; c <= 5; c++ {
			t.Run(fmt.Sprintf("ServiceCount=%d", c), func(t *testing.T) {
				testNewServiceFanoutSuccessLast(t, c)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		for c := 1; c <= 5; c++ {
			t.Run(fmt.Sprintf("ServiceCount=%d", c), func(t *testing.T) {
				testNewServiceFanoutTimeout(t, c)
			})
		}
	})

	t.Run("AllFail", func(t *testing.T) {
		for c := 1; c <= 5; c++ {
			t.Run(fmt.Sprintf("ServiceCount=%d", c), func(t *testing.T) {
				testNewServiceFanoutAllFail(t, c)
			})
		}
	})
}

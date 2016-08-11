package httppool

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestRequestTask(t *testing.T) {
	assert := assert.New(t)

	expectedRequest := MustNewRequest("GET", "http://example.com/")

	consumerCalled := false
	expectedConsumer := Consumer(func(*http.Response, *http.Request) {
		consumerCalled = true
	})

	task := RequestTask(expectedRequest, expectedConsumer)
	if actualRequest, actualConsumer, err := task(); assert.Nil(err) {
		assert.Equal(expectedRequest, actualRequest)
		assert.NotNil(actualConsumer)
		actualConsumer(nil, nil)
		assert.True(consumerCalled)
	}
}

func TestPerishableTaskGoneBad(t *testing.T) {
	assert := assert.New(t)

	expectedRequest := MustNewRequest("GET", "http://perishable.com/bad")

	consumer := func(*http.Response, *http.Request) {
		assert.Fail("The consumer should not have been called")
	}

	task := func() (*http.Request, Consumer, error) {
		assert.Fail("The delegate task should not be called when expiry time is reached")
		return expectedRequest, consumer, nil
	}

	expiry := time.Now().Add(-1 * time.Hour)
	perishable := PerishableTask(expiry, task)

	actualRequest, actualConsumer, err := perishable()
	assert.Nil(actualRequest)
	assert.Nil(actualConsumer)
	assert.Equal(ErrorTaskExpired, err)
}

func TestPerishableTaskStillFresh(t *testing.T) {
	assert := assert.New(t)

	expectedRequest := MustNewRequest("GET", "http://perishable.com/fresh")

	consumerCalled := false
	consumer := func(*http.Response, *http.Request) {
		consumerCalled = true
	}

	taskCalled := false
	task := func() (*http.Request, Consumer, error) {
		taskCalled = true
		return expectedRequest, consumer, nil
	}

	expiry := time.Now().Add(1 * time.Hour)
	perishable := PerishableTask(expiry, task)

	actualRequest, actualConsumer, err := perishable()
	assert.Equal(expectedRequest, actualRequest)
	assert.Nil(err)
	assert.True(taskCalled)
	if assert.NotNil(actualConsumer) {
		actualConsumer(nil, actualRequest)
		assert.True(consumerCalled)
	}
}

func TestFilteredTaskAccepted(t *testing.T) {
	assert := assert.New(t)

	expectedRequest := MustNewRequest("GET", "http://perishable.com/accepted")

	consumerCalled := false
	consumer := func(*http.Response, *http.Request) {
		consumerCalled = true
	}

	taskCalled := false
	task := func() (*http.Request, Consumer, error) {
		taskCalled = true
		return expectedRequest, consumer, nil
	}

	mockRequestFilter := &mockRequestFilter{}
	mockRequestFilter.On("Accept", expectedRequest).Return(true).Once()

	filtered := FilteredTask(mockRequestFilter, task)
	actualRequest, actualConsumer, err := filtered()
	assert.Equal(expectedRequest, actualRequest)
	assert.True(taskCalled)
	assert.Nil(err)
	if assert.NotNil(actualConsumer) {
		actualConsumer(nil, actualRequest)
		assert.True(consumerCalled)
	}

	mockRequestFilter.AssertExpectations(t)
}

func TestFilteredTaskRejected(t *testing.T) {
	assert := assert.New(t)

	expectedRequest := MustNewRequest("GET", "http://perishable.com/rejected")

	consumer := func(*http.Response, *http.Request) {
		assert.Fail("The consumer should not have been called")
	}

	taskCalled := false
	task := func() (*http.Request, Consumer, error) {
		taskCalled = true
		return expectedRequest, consumer, nil
	}

	mockRequestFilter := &mockRequestFilter{}
	mockRequestFilter.On("Accept", expectedRequest).Return(false).Once()

	filtered := FilteredTask(mockRequestFilter, task)
	actualRequest, actualConsumer, err := filtered()
	assert.Nil(actualRequest)
	assert.True(taskCalled)
	assert.Equal(ErrorTaskFiltered, err)
	assert.Nil(actualConsumer)

	mockRequestFilter.AssertExpectations(t)
}

func TestFilteredTaskDelegateNilRequest(t *testing.T) {
	assert := assert.New(t)

	consumer := func(*http.Response, *http.Request) {
		assert.Fail("The consumer should not have been called")
	}

	taskCalled := false
	task := func() (*http.Request, Consumer, error) {
		taskCalled = true
		return nil, consumer, nil
	}

	mockRequestFilter := &mockRequestFilter{}

	filtered := FilteredTask(mockRequestFilter, task)
	actualRequest, actualConsumer, err := filtered()
	assert.Nil(actualRequest)
	assert.True(taskCalled)
	assert.Nil(err)
	assert.Nil(actualConsumer)

	mockRequestFilter.AssertExpectations(t)
}

func TestFilteredTaskDelegateError(t *testing.T) {
	assert := assert.New(t)

	expectedError := errors.New("expected")
	expectedRequest := MustNewRequest("GET", "http://perishable.com/delegate/error")
	consumer := func(*http.Response, *http.Request) {
		assert.Fail("The consumer should not have been called")
	}

	taskCalled := false
	task := func() (*http.Request, Consumer, error) {
		taskCalled = true
		return expectedRequest, consumer, expectedError
	}

	mockRequestFilter := &mockRequestFilter{}

	filtered := FilteredTask(mockRequestFilter, task)
	actualRequest, actualConsumer, err := filtered()
	assert.Nil(actualRequest)
	assert.True(taskCalled)
	assert.Equal(expectedError, err)
	assert.Nil(actualConsumer)

	mockRequestFilter.AssertExpectations(t)
}

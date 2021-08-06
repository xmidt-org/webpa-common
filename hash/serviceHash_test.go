package hash

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/concurrent"
)

const (
	testKey = "this value does not matter"
)

var (
	successMessage1 = "first success message"
	successMessage2 = "second success message"
	errorMessage1   = "first error message"
	errorMessage2   = "second error message"
)

type errorServiceHash string

func (hash errorServiceHash) Get([]byte) (string, error) {
	return "", errors.New(string(hash))
}

type successServiceHash string

func (hash successServiceHash) Get([]byte) (string, error) {
	return string(hash), nil
}

func TestServiceHashHolderUninitialized(t *testing.T) {
	assert := assert.New(t)
	var holder ServiceHashHolder
	assert.False(holder.Connected())

	value, err := holder.Get([]byte(testKey))
	assert.Equal(ServiceHashHolderUninitialized, err)
	assert.Empty(value)
}

func TestServiceHashHolderGet(t *testing.T) {
	assert := assert.New(t)
	var gets = []struct {
		value        *string
		errorMessage *string
	}{
		{&successMessage1, nil},
		{nil, &errorMessage1},
		{&successMessage2, nil},
		{nil, &errorMessage2},
		{&successMessage2, nil},
		{&successMessage1, nil},
		{nil, &errorMessage2},
	}

	var holder ServiceHashHolder
	assert.False(holder.Connected())

	for _, record := range gets {
		if record.value != nil {
			holder.Update(successServiceHash(*record.value))
			actual, err := holder.Get([]byte(testKey))
			assert.Equal(*record.value, actual)
			assert.Nil(err)
		} else {
			holder.Update(errorServiceHash(*record.errorMessage))
			actual, err := holder.Get([]byte(testKey))
			assert.Empty(actual)
			assert.NotNil(err)
		}
	}
}

func TestServiceHashHolderConcurrent(t *testing.T) {
	assert := assert.New(t)

	var holder ServiceHashHolder
	assert.False(holder.Connected())

	available := []ServiceHash{
		successServiceHash(successMessage1),
		errorServiceHash(errorMessage1),
		successServiceHash(successMessage2),
		errorServiceHash(errorMessage2),
	}

	const getCount int = 100
	updates := make(chan ServiceHash, getCount)
	for index := 0; index < getCount; index++ {
		updates <- available[index%len(available)]
	}

	close(updates)
	const routineCount int = 3
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(routineCount)
	for index := 0; index < routineCount; index++ {
		go func() {
			defer waitGroup.Done()
			holder.Get([]byte(testKey))

			for update := range updates {
				holder.Update(update)
				holder.Get([]byte(testKey))
			}
		}()
	}

	assert.True(concurrent.WaitTimeout(waitGroup, 15*time.Second))
}

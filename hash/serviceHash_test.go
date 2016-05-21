package hash

import (
	"errors"
	"github.com/Comcast/webpa-common/concurrent"
	"sync"
	"testing"
	"time"
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
	var holder ServiceHashHolder
	value, err := holder.Get([]byte(testKey))

	if len(value) > 0 {
		t.Error("Get() should return an empty string when uninitialized")
	}

	if err != ServiceHashHolderUninitialized {
		t.Error("Get() should return ServiceHashHolderUninitialized when uninitialized")
	}
}

func TestServiceHashHolderGet(t *testing.T) {
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
	for _, record := range gets {
		if record.value != nil {
			holder.Update(successServiceHash(*record.value))
			actual, err := holder.Get([]byte(testKey))
			if *record.value != actual {
				t.Errorf("Expected value %s, but got %s", *record.value, actual)
			}

			if err != nil {
				t.Error("Get() should not have returned an error, but instead returned %v", err)
			}
		} else {
			holder.Update(errorServiceHash(*record.errorMessage))
			actual, err := holder.Get([]byte(testKey))
			if len(actual) > 0 {
				t.Errorf("Get() should have returned an empty string, but instead returned %s", actual)
			}

			if err == nil {
				t.Error("Get() should have returned an error for a successful hash")
			}
		}
	}
}

func TestServiceHashHolderConcurrent(t *testing.T) {
	var holder ServiceHashHolder

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

	if !concurrent.WaitTimeout(waitGroup, 15*time.Second) {
		t.Fatal("Failed to finish within the timeout")
	}
}

package concurrent

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestKeyValueEmpty(t *testing.T) {
	assert := assert.New(t)
	testWaitGroup := &sync.WaitGroup{}

	keyValue := NewKeyValue()
	if !assert.NotNil(keyValue) {
		return
	}

	assert.Nil(<-keyValue.Get(1))

	testWaitGroup.Add(1)
	keyValue.Do(
		KeyValueOperationFunc(func(storage KeyValueStorage) {
			defer testWaitGroup.Done()
			assert.Equal(0, len(storage))
		}),
	)
	testWaitGroup.Wait()

	for _ = range <-keyValue.Keys() {
		t.Error("Should not have received any keys in an empty KeyValue")
	}

	for _ = range <-keyValue.Values() {
		t.Error("Should not have received any values in an empty KeyValue")
	}
}

func TestKeyValueBasics(t *testing.T) {
	assert := assert.New(t)
	testWaitGroup := &sync.WaitGroup{}

	keyValue := NewKeyValue()
	if !assert.NotNil(keyValue) {
		return
	}

	keyValue.Add(1, "one")
	keyValue.Add(2, "two")
	keyValue.Add(3, "three")
	time.Sleep(50 * time.Millisecond)

	assert.Equal("one", <-keyValue.Get(1))
	assert.Equal("two", <-keyValue.Get(2))
	assert.Equal("three", <-keyValue.Get(3))
	assert.Nil(<-keyValue.Get(-192347192874918273))

	testWaitGroup.Add(1)
	keyValue.Do(
		KeyValueOperationFunc(func(storage KeyValueStorage) {
			defer testWaitGroup.Done()
			assert.Equal(
				KeyValueStorage(map[interface{}]interface{}{
					1: "one",
					2: "two",
					3: "three",
				}),
				storage,
			)
		}),
	)
	testWaitGroup.Wait()

	{
		expectedKeys := []int{1, 2, 3}
		sort.Ints(expectedKeys)
		actualKeys := []int{}
		for key := range <-keyValue.Keys() {
			actualKeys = append(actualKeys, key.(int))
		}

		sort.Ints(actualKeys)
		assert.Equal(expectedKeys, actualKeys)
	}

	{
		expectedValues := []string{"one", "two", "three"}
		sort.Strings(expectedValues)
		actualValues := []string{}
		for value := range <-keyValue.Values() {
			actualValues = append(actualValues, value.(string))
		}

		sort.Strings(actualValues)
		assert.Equal(expectedValues, actualValues)
	}

	keyValue.Delete(1, 2)
	time.Sleep(50 * time.Millisecond)

	assert.Nil(<-keyValue.Get(1))
	assert.Nil(<-keyValue.Get(2))
	assert.Equal("three", <-keyValue.Get(3))

	testWaitGroup.Add(1)
	keyValue.Do(
		KeyValueOperationFunc(func(storage KeyValueStorage) {
			defer testWaitGroup.Done()
			assert.Equal(
				KeyValueStorage(map[interface{}]interface{}{
					3: "three",
				}),
				storage,
			)
		}),
	)
	testWaitGroup.Wait()

	{
		expectedKeys := []int{3}
		actualKeys := []int{}
		for key := range <-keyValue.Keys() {
			actualKeys = append(actualKeys, key.(int))
		}

		assert.Equal(expectedKeys, actualKeys)
	}

	{
		expectedValues := []string{"three"}
		actualValues := []string{}
		for value := range <-keyValue.Values() {
			actualValues = append(actualValues, value.(string))
		}

		assert.Equal(expectedValues, actualValues)
	}
}

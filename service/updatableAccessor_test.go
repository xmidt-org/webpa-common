package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testUpdatableAccessorGetUpdate(t *testing.T) {
	var (
		assert   = assert.New(t)
		accessor = new(mockAccessor)

		u = new(UpdatableAccessor)
	)

	accessor.On("Get", mock.AnythingOfType("[]uint8")).Return("expected", error(nil)).Once()

	instance, err := u.Get([]byte("some key"))
	assert.Empty(instance)
	assert.Error(err)

	u.Update(accessor)
	instance, err = u.Get([]byte("some key"))
	assert.Equal("expected", instance)
	assert.NoError(err)

	u.Update(nil)
	instance, err = u.Get([]byte("some key"))
	assert.Empty(instance)
	assert.Error(err)

	accessor.AssertExpectations(t)
}

func testUpdatableAccessorConsume(t *testing.T) {
	const retries = 2

	var (
		assert       = assert.New(t)
		subscription = new(mockSubscription)
		accessor1    = new(mockAccessor)
		accessor2    = new(mockAccessor)

		updates = make(chan Accessor, 10)
		stopped = make(chan struct{})

		u = new(UpdatableAccessor)
	)

	subscription.On("Updates").Return((<-chan Accessor)(updates))
	subscription.On("Stopped").Return((<-chan struct{})(stopped))

	accessor1.On("Get", mock.AnythingOfType("[]uint8")).Return("accessor1", error(nil)).Once()
	accessor2.On("Get", mock.AnythingOfType("[]uint8")).Return("accessor2", error(nil)).Once()

	u.Consume(subscription)
	instance, err := u.Get([]byte("some key"))
	assert.Empty(instance)
	assert.Error(err)

	updates <- accessor1
	for r := 0; r < retries; r++ {
		time.Sleep(250 * time.Millisecond)
		if instance, err = u.Get([]byte("some key")); instance == "accessor1" && err == nil {
			// passed
			break
		} else if r == (retries - 1) {
			assert.Fail("No update occurred")
		}
	}

	updates <- accessor2
	for r := 0; r < retries; r++ {
		time.Sleep(250 * time.Millisecond)
		if instance, err = u.Get([]byte("some key")); instance == "accessor2" && err == nil {
			// passed
			break
		} else if r == (retries - 1) {
			assert.Fail("No update occurred")
		}
	}

	close(stopped)
	subscription.AssertExpectations(t)
}

func TestUpdatableAccessor(t *testing.T) {
	t.Run("GetUpdate", testUpdatableAccessorGetUpdate)
	t.Run("Consume", testUpdatableAccessorConsume)
}

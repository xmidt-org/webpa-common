package semaphore

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleMutex() {
	const routineCount = 5

	var (
		s     = Mutex()
		wg    = new(sync.WaitGroup)
		value int
	)

	wg.Add(routineCount)
	for i := 0; i < routineCount; i++ {
		go func() {
			defer wg.Done()
			defer s.Release()
			s.Acquire()
			value++
			fmt.Println(value)
		}()
	}

	wg.Wait()

	// Unordered output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

func testNewInvalidCount(t *testing.T) {
	for _, c := range []int{0, -1} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			assert.Panics(t, func() {
				New(c)
			})
		})
	}
}

func testNewValidCount(t *testing.T) {
	for _, c := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			s := New(c)
			assert.NotNil(t, s)
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("InvalidCount", testNewInvalidCount)
	t.Run("ValidCount", testNewValidCount)
}

func testTryAcquire(t *testing.T, s Interface, totalCount int) {
	assert := assert.New(t)
	for i := 0; i < totalCount; i++ {
		assert.True(s.TryAcquire())
	}

	assert.False(s.TryAcquire())
	s.Release()
	assert.True(s.TryAcquire())
	assert.False(s.TryAcquire())
}

func testAcquire(t *testing.T, s Interface, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.Acquire()
		}()

		select {
		case <-done:
			// passing
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// post condition: no point continuing if this fails
	require.False(s.TryAcquire())

	var (
		ready    = make(chan struct{})
		acquired = make(chan struct{})
	)

	go func() {
		defer close(acquired)
		close(ready)
		s.Acquire() // this should now block
	}()

	select {
	case <-ready:
		// passing
		require.False(s.TryAcquire())
		s.Release()
	case <-time.After(time.Second):
		require.FailNow("Unable to spawn acquire goroutine")
	}

	select {
	case <-acquired:
		require.False(s.TryAcquire())
	case <-time.After(time.Second):
		require.FailNow("Acquire blocked unexpectedly")
	}

	s.Release()
	assert.True(s.TryAcquire())
}

func testAcquireWait(t *testing.T, s Interface, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		timer   = make(chan time.Time)
	)

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		result := make(chan error)
		go func() {
			result <- s.AcquireWait(timer)
		}()

		select {
		case err := <-result:
			assert.NoError(err)
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// post condition: no point continuing if this fails
	require.False(s.TryAcquire())

	var (
		ready  = make(chan struct{})
		result = make(chan error)
	)

	go func() {
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		timer <- time.Time{}
	case <-time.After(time.Second):
		require.FailNow("Unable to spawn acquire goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ErrTimeout, err)
	case <-time.After(time.Second):
		require.FailNow("AcquireWait blocked unexpectedly")
	}
}

func testAcquireCtx(t *testing.T, s Interface, totalCount int) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		ctx, cancel = context.WithCancel(context.Background())
	)

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		result := make(chan error)
		go func() {
			result <- s.AcquireCtx(ctx)
		}()

		select {
		case err := <-result:
			assert.NoError(err)
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// post condition: no point continuing if this fails
	require.False(s.TryAcquire())

	var (
		ready  = make(chan struct{})
		result = make(chan error)
	)

	go func() {
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		cancel()
	case <-time.After(time.Second):
		require.FailNow("Unable to spawn acquire goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ctx.Err(), err)
	case <-time.After(time.Second):
		require.FailNow("AcquireWait blocked unexpectedly")
	}
}

func TestSemaphore(t *testing.T) {
	for _, c := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", c), func(t *testing.T) {
			t.Run("TryAcquire", func(t *testing.T) {
				testTryAcquire(t, New(c), c)
			})

			t.Run("Acquire", func(t *testing.T) {
				testAcquire(t, New(c), c)
			})

			t.Run("AcquireWait", func(t *testing.T) {
				testAcquireWait(t, New(c), c)
			})

			t.Run("AcquireCtx", func(t *testing.T) {
				testAcquireCtx(t, New(c), c)
			})
		})
	}
}

func TestMutex(t *testing.T) {
	t.Run("TryAcquire", func(t *testing.T) {
		testTryAcquire(t, Mutex(), 1)
	})

	t.Run("Acquire", func(t *testing.T) {
		testAcquire(t, Mutex(), 1)
	})

	t.Run("AcquireWait", func(t *testing.T) {
		testAcquireWait(t, Mutex(), 1)
	})

	t.Run("AcquireCtx", func(t *testing.T) {
		testAcquireCtx(t, Mutex(), 1)
	})
}

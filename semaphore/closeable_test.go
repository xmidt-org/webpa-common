package semaphore

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewCloseableInvalidCount(t *testing.T) {
	for _, c := range []int{0, -1} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			assert.Panics(t, func() {
				NewCloseable(c)
			})
		})
	}
}

func testNewCloseableValidCount(t *testing.T) {
	for _, c := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			s := NewCloseable(c)
			assert.NotNil(t, s)
		})
	}
}

func TestNewCloseable(t *testing.T) {
	t.Run("InvalidCount", testNewCloseableInvalidCount)
	t.Run("ValidCount", testNewCloseableValidCount)
}

func testCloseableTryAcquire(t *testing.T, cs Closeable, totalCount int) {
	assert := assert.New(t)
	for i := 0; i < totalCount; i++ {
		assert.True(cs.TryAcquire())
	}

	assert.False(cs.TryAcquire())
	assert.NoError(cs.Release())
	assert.True(cs.TryAcquire())
	assert.False(cs.TryAcquire())

	assert.NoError(cs.Release())
	// nolint: typecheck
	assert.NoError(cs.Close())
	assert.False(cs.TryAcquire())
	// nolint: typecheck
	assert.Equal(ErrClosed, cs.Close())
	assert.Equal(ErrClosed, cs.Release())
}

func testCloseableAcquireSuccess(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		done := make(chan struct{})
		go func() {
			defer close(done)
			cs.Acquire()
		}()

		select {
		case <-done:
			// passing
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	var (
		ready    = make(chan struct{})
		acquired = make(chan struct{})
	)

	go func() {
		defer close(acquired)
		close(ready)
		cs.Acquire() // this should now block
	}()

	select {
	case <-ready:
		// passing
		require.False(cs.TryAcquire())
		cs.Release()
	case <-time.After(time.Second):
		require.FailNow("Unable to spawn acquire goroutine")
	}

	select {
	case <-acquired:
		require.False(cs.TryAcquire())
	case <-time.After(time.Second):
		require.FailNow("Acquire blocked unexpectedly")
	}

	assert.NoError(cs.Release())
	assert.True(cs.TryAcquire())
	assert.NoError(cs.Release())
}

func testCloseableAcquireClose(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		acquiredAll = make(chan struct{})
		results     = make(chan error, totalCount)
		closeWait   = make(chan struct{})
	)

	// nolint: typecheck
	defer cs.Close()

	go func() {
		defer close(acquiredAll)
		for i := 0; i < totalCount; i++ {
			assert.NoError(cs.Acquire())
		}
	}()

	select {
	case <-acquiredAll:
		// passing
	case <-time.After(5 * time.Second):
		assert.FailNow("Unable to acquire all resources")
	}

	// block multiple routines waiting to acquire the semaphore
	for i := 0; i < totalCount; i++ {
		ready := make(chan struct{})
		go func() {
			close(ready)
			results <- cs.Acquire()
		}()

		select {
		case <-ready:
			// passing
		case <-time.After(time.Second):
			assert.FailNow("Failed to spawn Acquire goroutine")
		}
	}

	go func() {
		defer close(closeWait)
		// nolint: typecheck
		<-cs.Closed()
	}()

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	// nolint: typecheck
	assert.NoError(cs.Close())
	for i := 0; i < totalCount; i++ {
		select {
		case err := <-results:
			assert.Equal(ErrClosed, err)
		case <-time.After(5 * time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	select {
	case <-closeWait:
		assert.False(cs.TryAcquire())
		// nolint: typecheck
		assert.Equal(ErrClosed, cs.Close())
		assert.Equal(ErrClosed, cs.Acquire())
		assert.Equal(ErrClosed, cs.Release())

	case <-time.After(5 * time.Second):
		assert.FailNow("Closed channel did not get signaled")
	}
}

func testCloseableAcquireWaitSuccess(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		timer   = make(chan time.Time)
	)

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		result := make(chan error)
		go func() {
			result <- cs.AcquireWait(timer)
		}()

		select {
		case err := <-result:
			assert.NoError(err)
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// nolint: typecheck
	defer cs.Close()

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	var (
		ready  = make(chan struct{})
		result = make(chan error)
	)

	go func() {
		close(ready)
		result <- cs.AcquireWait(timer)
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

func testCloseableAcquireWaitClose(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		timer   = make(chan time.Time)

		acquiredAll = make(chan struct{})
		results     = make(chan error, totalCount)
		closeWait   = make(chan struct{})
	)

	// nolint: typecheck
	defer cs.Close()

	go func() {
		defer close(acquiredAll)
		for i := 0; i < totalCount; i++ {
			assert.NoError(cs.Acquire())
		}
	}()

	select {
	case <-acquiredAll:
		// passing
	case <-time.After(5 * time.Second):
		assert.FailNow("Unable to acquire all resources")
	}

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		ready := make(chan struct{})
		go func() {
			close(ready)
			results <- cs.AcquireWait(timer)
		}()

		select {
		case <-ready:
			// passing
		case <-time.After(5 * time.Second):
			assert.FailNow("Failed to spawn AcquireWait goroutine")
		}
	}

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	go func() {
		defer close(closeWait)
		// nolint: typecheck
		<-cs.Closed()
	}()

	// nolint: typecheck
	assert.NoError(cs.Close())
	for i := 0; i < totalCount; i++ {
		select {
		case err := <-results:
			assert.Equal(ErrClosed, err)
		case <-time.After(5 * time.Second):
			assert.FailNow("AcquireWait blocked unexpectedly")
		}
	}

	select {
	case <-closeWait:
		assert.False(cs.TryAcquire())
		// nolint: typecheck
		assert.Equal(ErrClosed, cs.Close())
		assert.Equal(ErrClosed, cs.Acquire())
		assert.Equal(ErrClosed, cs.Release())

	case <-time.After(5 * time.Second):
		assert.FailNow("Closed channel did not get signaled")
	}
}

func testCloseableAcquireCtxSuccess(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		result := make(chan error)
		go func() {
			result <- cs.AcquireCtx(ctx)
		}()

		select {
		case err := <-result:
			assert.NoError(err)
		case <-time.After(time.Second):
			assert.FailNow("Acquire blocked unexpectedly")
		}
	}

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	var (
		ready  = make(chan struct{})
		result = make(chan error)
	)

	go func() {
		close(ready)
		result <- cs.AcquireCtx(ctx)
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

func testCloseableAcquireCtxClose(t *testing.T, cs Closeable, totalCount int) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		ctx, cancel = context.WithCancel(context.Background())

		acquiredAll = make(chan struct{})
		results     = make(chan error, totalCount)
		closeWait   = make(chan struct{})
	)

	defer cancel()

	go func() {
		defer close(acquiredAll)
		for i := 0; i < totalCount; i++ {
			assert.NoError(cs.Acquire())
		}
	}()

	select {
	case <-acquiredAll:
		// passing
	case <-time.After(5 * time.Second):
		assert.FailNow("Unable to acquire all resources")
	}

	// acquire all the things!
	for i := 0; i < totalCount; i++ {
		ready := make(chan struct{})
		go func() {
			close(ready)
			results <- cs.AcquireCtx(ctx)
		}()

		select {
		case <-ready:
			// passing
		case <-time.After(5 * time.Second):
			assert.FailNow("Could not spawn AcquireCtx goroutine")
		}
	}

	// post condition: no point continuing if this fails
	require.False(cs.TryAcquire())

	go func() {
		defer close(closeWait)
		// nolint: typecheck
		<-cs.Closed()
	}()

	// nolint: typecheck
	assert.NoError(cs.Close())
	for i := 0; i < totalCount; i++ {
		select {
		case err := <-results:
			assert.Equal(ErrClosed, err)
		case <-time.After(5 * time.Second):
			assert.FailNow("AcquireCtx blocked unexpectedly")
		}
	}

	select {
	case <-closeWait:
		assert.False(cs.TryAcquire())
		// nolint: typecheck
		assert.Equal(ErrClosed, cs.Close())
		assert.Equal(ErrClosed, cs.Acquire())
		assert.Equal(ErrClosed, cs.Release())

	case <-time.After(5 * time.Second):
		assert.FailNow("Closed channel did not get signaled")
	}
}

func TestCloseable(t *testing.T) {
	for _, c := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("count=%d", c), func(t *testing.T) {
			t.Run("TryAcquire", func(t *testing.T) {
				testCloseableTryAcquire(t, NewCloseable(c), c)
			})

			t.Run("Acquire", func(t *testing.T) {
				t.Run("Success", func(t *testing.T) {
					testCloseableAcquireSuccess(t, NewCloseable(c), c)
				})

				t.Run("Close", func(t *testing.T) {
					testCloseableAcquireClose(t, NewCloseable(c), c)
				})
			})

			t.Run("AcquireWait", func(t *testing.T) {
				t.Run("Success", func(t *testing.T) {
					testCloseableAcquireWaitSuccess(t, NewCloseable(c), c)
				})

				t.Run("Close", func(t *testing.T) {
					testCloseableAcquireWaitClose(t, NewCloseable(c), c)
				})
			})

			t.Run("AcquireCtx", func(t *testing.T) {
				t.Run("Success", func(t *testing.T) {
					testCloseableAcquireCtxSuccess(t, NewCloseable(c), c)
				})

				t.Run("Close", func(t *testing.T) {
					testCloseableAcquireCtxClose(t, NewCloseable(c), c)
				})
			})
		})
	}
}

func TestCloseableMutex(t *testing.T) {
	t.Run("TryAcquire", func(t *testing.T) {
		testCloseableTryAcquire(t, CloseableMutex(), 1)
	})

	t.Run("Acquire", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			testCloseableAcquireSuccess(t, CloseableMutex(), 1)
		})

		t.Run("Close", func(t *testing.T) {
			testCloseableAcquireClose(t, CloseableMutex(), 1)
		})
	})

	t.Run("AcquireWait", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			testCloseableAcquireWaitSuccess(t, CloseableMutex(), 1)
		})

		t.Run("Close", func(t *testing.T) {
			testCloseableAcquireWaitClose(t, CloseableMutex(), 1)
		})
	})

	t.Run("AcquireCtx", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			testCloseableAcquireCtxSuccess(t, CloseableMutex(), 1)
		})

		t.Run("Close", func(t *testing.T) {
			testCloseableAcquireCtxClose(t, CloseableMutex(), 1)
		})
	})
}

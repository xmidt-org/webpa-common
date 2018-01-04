package server

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testSignalWaitBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		logger  = logging.NewTestLogger(nil, t)
		signals = make(chan os.Signal)

		started  = new(sync.WaitGroup)
		finished = make(chan os.Signal)
	)

	defer close(signals)
	started.Add(1)
	go func() {
		started.Done()
		finished <- SignalWait(logger, signals, os.Kill)
	}()

	started.Wait()

	signals <- os.Interrupt
	select {
	case <-finished:
		assert.Fail("os.Interrupt should not have ended SignalWait")
	default:
		// passing
	}

	signals <- os.Kill
	select {
	case actual := <-finished:
		assert.Equal(os.Kill, actual)
	case <-time.After(10 * time.Second):
		assert.Fail("SignalWait did not complete within the timeout")
	}
}

func testSignalWaitForever(t *testing.T) {
	var (
		assert  = assert.New(t)
		logger  = logging.NewTestLogger(nil, t)
		signals = make(chan os.Signal)

		started  = new(sync.WaitGroup)
		finished = make(chan os.Signal)
	)

	started.Add(1)
	go func() {
		started.Done()
		finished <- SignalWait(logger, signals)
	}()

	started.Wait()
	for _, s := range []os.Signal{os.Kill, os.Interrupt} {
		signals <- s
		select {
		case <-finished:
			assert.Fail("SignalWait should not have finished")
		default:
			// passing
		}
	}

	close(signals)
	select {
	case actual := <-finished:
		assert.Nil(actual)
	case <-time.After(10 * time.Second):
		assert.Fail("SignalWait did not complete within the timeout")
	}
}

func TestSignalWait(t *testing.T) {
	t.Run("Basic", testSignalWaitBasic)
	t.Run("WaitForever", testSignalWaitForever)
}

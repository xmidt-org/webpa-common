package conlimiter

import (
	//"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type MyMockedConnection struct {
	mock.Mock
	CloseCalled int32
}

func (m *MyMockedConnection) Close() error {
	atomic.AddInt32(&m.CloseCalled, 1)
	return nil
}

func (m *MyMockedConnection) Read(b []byte) (int, error) {
	return 0, nil
}

func (m *MyMockedConnection) Write(b []byte) (int, error) {
	return 0, nil
}

func (m *MyMockedConnection) LocalAddr() net.Addr {
	return nil
}

func (m *MyMockedConnection) RemoteAddr() net.Addr {
	return nil
}

func (m *MyMockedConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *MyMockedConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MyMockedConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestLimit(t *testing.T) {
	testObj := new(MyMockedConnection)
	assert := assert.New(t)

	s := &http.Server{Addr: ":61234"}
	cl := &ConLimiter{Max: 10}
	cl.Limit(s)

	var zero int32 = 0
	var one int32 = 1

	for i := 0; i < 10; i++ {
		(s.ConnState)(testObj, http.StateNew)
		assert.Equal(zero, atomic.LoadInt32(&testObj.CloseCalled), "Expecting no call.")
	}
	for i := 0; i < 10; i++ {
		(s.ConnState)(testObj, http.StateNew)
		assert.Equal(one, atomic.LoadInt32(&testObj.CloseCalled), "Expecting 1 call.")
		atomic.StoreInt32(&testObj.CloseCalled, 0)
		(s.ConnState)(testObj, http.StateClosed)
		assert.Equal(zero, atomic.LoadInt32(&testObj.CloseCalled), "Expecting no call.")
	}
	for i := 0; i < 10; i++ {
		(s.ConnState)(testObj, http.StateHijacked)
		assert.Equal(zero, atomic.LoadInt32(&testObj.CloseCalled), "Expecting no call.")
	}

	assert.Equal(zero, atomic.LoadInt32(&testObj.CloseCalled), "Expecting it to be 0.")
	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go (func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				(s.ConnState)(testObj, http.StateNew)
				(s.ConnState)(testObj, http.StateClosed)
			}
		})()
		go (func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				(s.ConnState)(testObj, http.StateNew)
				(s.ConnState)(testObj, http.StateActive)
				(s.ConnState)(testObj, http.StateClosed)
			}
		})()
		go (func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				(s.ConnState)(testObj, http.StateNew)
				(s.ConnState)(testObj, http.StateActive)
				(s.ConnState)(testObj, http.StateHijacked)
			}
		})()
	}
	wg.Wait()
	assert.Equal(zero, cl.current, "Expecting it to be 0.")
}

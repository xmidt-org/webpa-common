package server

import (
	"bytes"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"net/http"
	"testing"
)

const (
	serverName = "serverName"
)

func newTestLogger() (verify *bytes.Buffer, logger logging.Logger) {
	verify = new(bytes.Buffer)
	logger = &logging.LoggerWriter{verify}
	return
}

func assertBufferContains(assert *assert.Assertions, verify *bytes.Buffer, values ...string) {
	text := verify.String()
	for _, value := range values {
		assert.Contains(text, value)
	}
}

func assertErrorLog(assert *assert.Assertions, verify *bytes.Buffer, serverName string, errorLog *log.Logger) {
	if assert.NotNil(errorLog) {
		errorLog.Print("howdy!")
		assertBufferContains(assert, verify, serverName, "howdy!")
	}
}

func assertConnState(assert *assert.Assertions, verify *bytes.Buffer, connState func(net.Conn, http.ConnState)) {
	if assert.NotNil(connState) {
		conn1, conn2 := net.Pipe()
		defer conn1.Close()
		defer conn2.Close()

		connState(conn1, http.StateNew)
		assertBufferContains(assert, verify, conn1.LocalAddr().String(), http.StateNew.String())
	}
}

func TestNewErrorLog(t *testing.T) {
	var (
		assert         = assert.New(t)
		verify, logger = newTestLogger()
		errorLog       = NewErrorLog(serverName, logger)
	)

	assertErrorLog(assert, verify, serverName, errorLog)
}

func TestNewConnectionStateLogger(t *testing.T) {
	var (
		assert            = assert.New(t)
		verify, logger    = newTestLogger()
		connectionLogFunc = NewConnectionStateLogger(serverName, logger)
	)

	assertConnState(assert, verify, connectionLogFunc)
}

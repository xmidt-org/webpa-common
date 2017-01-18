package server

import (
	"bytes"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
)

const (
	serverName = "serverName"
)

func TestNewErrorLog(t *testing.T) {
	var (
		assert   = assert.New(t)
		output   bytes.Buffer
		logger   = logging.LoggerWriter{&output}
		errorLog = NewErrorLog(serverName, &logger)
	)

	errorLog.Print("howdy!")
	text := output.String()
	assert.Contains(text, serverName)
	assert.Contains(text, "howdy!")
}

func TestNewConnectionStateLogger(t *testing.T) {
	var (
		assert = assert.New(t)

		conn1, conn2      = net.Pipe()
		output            bytes.Buffer
		logger            = logging.LoggerWriter{&output}
		connectionLogFunc = NewConnectionStateLogger(serverName, &logger)
	)

	defer conn1.Close()
	defer conn2.Close()

	connectionLogFunc(conn1, http.StateNew)
	text := output.String()
	assert.Contains(text, conn1.LocalAddr().String())
	assert.Contains(text, http.StateNew.String())
}

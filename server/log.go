package server

import (
	"github.com/Comcast/webpa-common/logging"
	"log"
	"net"
	"net/http"
)

// NewErrorLog creates a new log.Logger appropriate for http.Server.ErrorLog
func NewErrorLog(serverName string, logger logging.Logger) *log.Logger {
	return log.New(&logging.ErrorWriter{logger}, serverName, log.LstdFlags|log.LUTC)
}

// NewConnectionStateLogger produces a function appropriate for http.Server.ConnState.
// The returned function will log debug statements for each state change.
func NewConnectionStateLogger(serverName string, logger logging.Logger) func(net.Conn, http.ConnState) {
	return func(connection net.Conn, connectionState http.ConnState) {
		logger.Debug(
			"[%s] [%s] -> %s",
			serverName,
			connection.LocalAddr().String(),
			connectionState,
		)
	}
}

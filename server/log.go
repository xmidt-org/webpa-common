package server

import (
	stdlibLog "log"
	"net"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

// NewErrorLog creates a new logging.Logger appropriate for http.Server.ErrorLog
func NewErrorLog(serverName string, logger log.Logger) *stdlibLog.Logger {
	return stdlibLog.New(
		log.NewStdlibAdapter(logger),
		serverName,
		stdlibLog.LstdFlags|stdlibLog.LUTC,
	)
}

// NewConnectionStateLogger produces a function appropriate for http.Server.ConnState.
// The returned function will log debug statements for each state change.
func NewConnectionStateLogger(serverName string, logger log.Logger) func(net.Conn, http.ConnState) {
	logger = logging.Debug(logger)
	return func(connection net.Conn, connectionState http.ConnState) {
		logger.Log(
			"serverName", serverName,
			"localAddress", connection.LocalAddr().String(),
			"state", connectionState,
		)
	}
}

package server

import (
	stdlibLog "log"
	"net"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"go.uber.org/zap"
)

// NewErrorLog creates a new logging.Logger appropriate for http.Server.ErrorLog
func NewErrorLog(serverName string, logger *zap.Logger) *stdlibLog.Logger {
	return stdlibLog.New(
		zap.NewStdLog(logger).Writer(),
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

package server

// NewErrorLog creates a new log.Logger appropriate for http.Server.ErrorLog
func NewErrorLog(logger Logger, serverName string) *log.Logger {
	return log.New(&ErrorWriter{logger}, serverName, log.LstdFlags|log.LUTC)
}

// NewConnectionStateLogger produces a function appropriate for http.Server.ConnState.
// The returned function will log debug statements for each state change.
func NewConnectionStateLogger(logger Logger, serverName string) func(net.Conn, http.ConnState) {
	return func(connection net.Conn, connectionState http.ConnState) {
		logger.Debugf(
			"[%s] [%s] -> %s",
			serverName,
			connection.LocalAddr().String(),
			connectionState,
		)
	}
}

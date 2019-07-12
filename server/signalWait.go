package server

import (
	"os"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

// SignalWait blocks until any of a set of signals is encountered.  The signal which caused this function
// to exit is returned.  A nil return indicates that the signals channel was closed.
//
// If no waitOn signals are supplied, this function will never return until the signals channel is closed.
//
// In all cases, the supplied logger is used to log information about signals that are ignored.
func SignalWait(logger log.Logger, signals <-chan os.Signal, waitOn ...os.Signal) os.Signal {
	filter := make(map[os.Signal]bool)
	for _, s := range waitOn {
		filter[s] = true
	}

	for s := range signals {
		if filter[s] {
			return s
		}

		logger.Log(logging.MessageKey(), "ignoring signal", "signal", s.String())
	}

	return nil
}

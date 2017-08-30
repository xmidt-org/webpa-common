package service

import (
	"errors"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

var (
	ErrorAlreadyRunning = errors.New("That subscription is already running")
	ErrorNotRunning     = errors.New("That subscription is not running")
)

// Subscription represents a specific sink for watch events.  The Listener function is notified
// with updated endpoints.
type Subscription struct {
	// Logger is the option Logger used by this subscription.  If not supplied, it defaults to logging.DefaultLogger().
	Logger log.Logger

	// Registrar is the service registration component used to create a Watch.
	Registrar Registrar

	// Listener is the sink for service endpoint updates.  This field is required, and must not
	// be changed concurrently with any methods of this type.
	//
	// This field can be set to UpdatableAccessor.Update.  That will simply update the accessor's
	// endpoints with every watch event.
	Listener func([]string)

	// Timeout is an optional interval used for fault tolerance in the face of network flapping.  If set
	// to a positive value, then updates will not be immediately dispatched to the Listener.  Rather, when an
	// update first occurs, a timer is started.  Within the timer interval, only the most recent update is kept.
	// When the timer elapses, the most recent update is dispatched to the Listener and this process starts over.
	Timeout time.Duration

	// After is an optional function which is used to produce a time channel for delays.  Setting this
	// field is only relevant if Timeout > 0.  If this field is nil, time.After is used.
	After func(time.Duration) <-chan time.Time

	mutex    sync.Mutex
	watch    Watch
	shutdown chan struct{}
}

// monitor is a goroutine that monitors the watch and dispatches updated endpoints
// to the Listener.
func (s *Subscription) monitor(watch Watch, shutdown <-chan struct{}) {
	var (
		logger = s.Logger
		delay  <-chan time.Time
		after  = s.After
	)

	if logger == nil {
		logger = logging.DefaultLogger()
	}

	var (
		errorLog = logging.Error(logger)
		infoLog  = logging.Info(logger)
	)

	if after == nil {
		after = time.After
	}

	defer func() {
		if r := recover(); r != nil {
			errorLog.Log(logging.MessageKey, "Subscription ending due to panic", "error", r)
		}

		// ensure that the cancellation logic runs in this case, since no explicit
		// call to Cancel may have happened, e.g. panic, the watch was closed, etc
		s.Cancel()
	}()

	endpoints := watch.Endpoints()
	errorLog.Log(logging.MessageKey, "Dispatching initial endpoints", "endpoints", endpoints)
	s.Listener(endpoints)
	endpoints = nil

	infoLog.Log(logging.MessageKey, "Monitoring subscription", "watch", watch)

	for {
		select {
		case <-shutdown:
			infoLog.Log(logging.MessageKey, "Subscription ending because it was cancelled")
			return

		case <-delay:
			delay = nil
			infoLog.Log(logging.MessageKey, "Dispatching updated endpoints after delay", "delay", s.Timeout, "endpoints", endpoints)
			s.Listener(endpoints)
			endpoints = nil

		case <-watch.Event():
			if watch.IsClosed() {
				infoLog.Log(logging.MessageKey, "Subscription ending because the watch was closed")
				return
			}

			endpoints = watch.Endpoints()

			if delay != nil {
				// there is a delay in effect, so just keep listening for updates
				infoLog.Log(logging.MessageKey, "Still waiting to dispatch updates", "delay", s.Timeout)
				continue
			}

			if s.Timeout > 0 {
				infoLog.Log(logging.MessageKey, "Waiting to dispatch updates", "delay", s.Timeout)
				delay = after(s.Timeout)
				continue
			}

			// there is no current delay and no Timeout configured,
			// so dispatch immediately
			infoLog.Log(logging.MessageKey, "Dispatching updated endpoints", "endpoints", endpoints)
			s.Listener(endpoints)
			endpoints = nil
		}
	}
}

// Run starts monitoring the watch for this subscription.  This method is idempotent, and returns
// ErrorAlreadyRunning if this instance is already running.
func (s *Subscription) Run() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.watch != nil {
		return ErrorAlreadyRunning
	}

	watch, err := s.Registrar.Watch()
	if err != nil {
		return err
	}

	s.watch = watch
	s.shutdown = make(chan struct{})
	go s.monitor(s.watch, s.shutdown)
	return nil
}

// Cancel stops monitoring the watch for this subscription.  This method is idempotent, and returns
// true to indicate that the subscription was cancelled.  If this subscription was not running,
// this method returns false.
func (s *Subscription) Cancel() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// close the shutdown channel first, so log messages accurately
	// reflect cancellation when applicable
	if s.shutdown != nil {
		close(s.shutdown)
		s.shutdown = nil
	}

	if s.watch != nil {
		s.watch.Close()
		s.watch = nil
		return nil
	}

	return ErrorNotRunning
}

package server

import (
	"context"
	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"net/http"
	"sync/atomic"
)

// serverable holds the fields of a server (e.g. ListenAndServe() and Shutdown())
type serverable struct {
	serve    func() error
	shutdown func(ctx context.Context) error
}

// restartServer holds on to the info to control the server and restartability
type restarterServer struct {
	listenFunc serverable
	stop       <-chan struct{}
	logger     log.Logger
	done       atomic.Value
}

// BeginRestartableServer starts a restartable server given a Serve function and a Shutdown func.
// The server will continue to run until a http.ErrServerClosed for someone closing the server or
// a msg sent to the stop chan. If some other error is thrown the server will just call the Serve
// func again.
//
// Creating a restartable server is simple and easy:
//	server := http.Server{}
//	BeginRestartableServer(server.ListenAndServe, server.Shutdown, nil, nil)
func BeginRestartableServer(serve func() error, shutdown func(ctx context.Context) error, stop <-chan struct{}, logger log.Logger) error {
	if serve == nil {
		return errors.New("serve func can't be null")
	}
	if shutdown == nil {
		return errors.New("shutdown func can't be null")
	}
	if logger == nil {
		logger = logging.DefaultLogger()
	}
	if stop == nil {
		logging.Warn(logger).Log(logging.MessageKey(), "won't be able to stop restartable server from here")
		stop = make(chan struct{}, 1)
	}
	server := restarterServer{
		listenFunc: serverable{
			serve:    serve,
			shutdown: shutdown,
		},
		stop:   stop,
		logger: logger,
	}
	server.done.Store(false)
	go server.do()
	return nil
}

func (server *restarterServer) do() {
	logging.Info(server.logger).Log(logging.MessageKey(), "starting restartable listenFunc")
	go func() {
		server.serve()
	}()
	<-server.stop
	server.done.Store(true)
	server.listenFunc.shutdown(context.Background())

	logging.Info(server.logger).Log(logging.MessageKey(), "restartable listenFunc is stopping")
}

func (server *restarterServer) serve() {
	if err := server.listenFunc.serve(); err != nil {
		logging.Error(server.logger).Log(logging.MessageKey(), "ListenAndServe failed; restarting", logging.ErrorKey(), err)
		// the restart logic
		if done, ok := server.done.Load().(bool); ok && done {
			// done
		} else if err == http.ErrServerClosed {
			// stop
		} else {
			server.serve()
		}
	}
}

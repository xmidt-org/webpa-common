package drain

import (
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/converter"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/schema"
)

type Start struct {
	Logger  log.Logger
	Drainer Interface
}

func (s *Start) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := s.Logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	if err := request.ParseForm(); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to parse form", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	var (
		decoder = schema.NewDecoder()
		job     Job
	)

	decoder.RegisterConverter(time.Duration(0), converter.Duration)
	if err := decoder.Decode(&job, request.Form); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to decode request", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if err := s.Drainer.Start(job); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to start drain job", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusConflict, err)
	}
}

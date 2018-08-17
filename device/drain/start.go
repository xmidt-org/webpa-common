package drain

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/converter"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/schema"
)

type Start struct {
	Drainer Interface
}

func (s *Start) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())
	if err := request.ParseForm(); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to parse form", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	var (
		decoder = schema.NewDecoder()
		input   Job
	)

	decoder.RegisterConverter(time.Duration(0), converter.Duration)
	if err := decoder.Decode(&input, request.Form); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to decode request", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	_, output, err := s.Drainer.Start(input)
	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to start drain job", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusConflict, err)
		return
	}

	if message, err := json.Marshal(output.ToMap()); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to marshal response", logging.ErrorKey(), err)
	} else {
		response.Header().Set("Content-Type", "application/json")
		response.Write(message)
	}
}

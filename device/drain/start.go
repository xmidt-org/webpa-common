package drain

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/schema"
	"github.com/xmidt-org/webpa-common/device/devicegate"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
	"github.com/xmidt-org/webpa-common/xhttp/converter"
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
		reqBody devicegate.FilterRequest
	)

	decoder.RegisterConverter(time.Duration(0), converter.Duration)
	if err := decoder.Decode(&input, request.Form); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to decode request", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	msgBytes, e := ioutil.ReadAll(request.Body)
	defer request.Body.Close()

	if e != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to read request body", logging.ErrorKey(), e)
		xhttp.WriteError(response, http.StatusBadRequest, e)
		return
	}

	if len(msgBytes) > 0 {
		if err := json.Unmarshal(msgBytes, &reqBody); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to unmarshal request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if len(reqBody.Key) > 0 && len(reqBody.Values) > 0 {
			fg := devicegate.FilterGate{FilterStore: make(devicegate.FilterStore)}
			fg.SetFilter(reqBody.Key, reqBody.Values)

			input.DrainFilter = &drainFilter{
				filter:        &fg,
				filterRequest: reqBody,
			}
		}
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

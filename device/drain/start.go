// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package drain

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/schema"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/device/devicegate"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"github.com/xmidt-org/webpa-common/v2/xhttp/converter"
	"go.uber.org/zap"
)

type Start struct {
	Drainer Interface
}

func (s *Start) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := sallust.Get(request.Context())
	if err := request.ParseForm(); err != nil {
		logger.Error("unable to parse form", zap.Error(err))
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
		logger.Error("unable to decode request", zap.Error(err))
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	msgBytes, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()

	if err != nil {
		logger.Error("unable to read request body", zap.Error(err))
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if len(msgBytes) > 0 {
		if err := json.Unmarshal(msgBytes, &reqBody); err != nil {
			logger.Error("unable to unmarshal request body", zap.Error(err))
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
		logger.Error("unable to start drain job", zap.Error(err))
		xhttp.WriteError(response, http.StatusConflict, err)
		return
	}

	if message, err := json.Marshal(output.ToMap()); err != nil {
		logger.Error("unable to marshal response", zap.Error(err))
	} else {
		response.Header().Set("Content-Type", "application/json")
		response.Write(message)
	}
}

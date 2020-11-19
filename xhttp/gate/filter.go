package gate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
)

// Filter is an http.Handler which controls what filters to filter a connection request by
type Filter struct {
	// Filter is the Filter that is part of a Connector to filter connection requests
	Filter device.Filter
}

type filterRequest struct {
	Key    string
	Values []string
}

func (f *Filter) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == "GET" {
		filters := f.Filter.GetFilters()
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(response, `{"filters": %v}`, filters)
	} else if method == "POST" || method == "PUT" || method == "DELETE" {
		var message filterRequest
		msgBytes, err := ioutil.ReadAll(request.Body)
		request.Body.Close()

		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not read request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		err = json.Unmarshal(msgBytes, &message)
		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not unmarshal request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if len(message.Key) == 0 {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no filter key found")
			xhttp.WriteErrorf(response, http.StatusBadRequest, "missing filter key")
			return
		}

		if len(message.Values) == 0 {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no filter values found")
			xhttp.WriteErrorf(response, http.StatusBadRequest, "missing filter values")
			return
		}

		if method == "POST" || method == "PUT" {
			if device.FilterMap != nil {
				_, ok := device.FilterMap[message.Key]

				if !ok {
					logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
					xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %v", message.Key, device.FilterMap)
					return
				}
			}

			f.Filter.EditFilters(message.Key, message.Values, true)

		} else if method == "DELETE" {
			f.Filter.EditFilters(message.Key, message.Values, false)
		}

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", f.Filter.GetFilters())

		response.WriteHeader(http.StatusOK)
	}
}

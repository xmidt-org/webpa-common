package devicegate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
)

type FilterHandler struct {
	Gate DeviceGate
}

func (fh *FilterHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == "GET" {
		filters := fh.Gate.GetFilters()
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

			_, ok := fh.Gate.AllowedFilters()[message.Key]

			if !ok {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
				xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %v", message.Key, fh.Gate.AllowedFilters())
				return
			}

			fh.Gate.SetFilter(message.Key, message.Values)

		} else if method == "DELETE" {
			fh.Gate.DeleteFilter(message.Key)
		}

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", fh.Gate.GetFilters())

		response.WriteHeader(http.StatusOK)
	}
}

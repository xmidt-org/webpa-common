package devicegate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
)

// FilterHandler is an http.Handler that can get, add, and delete filters from a devicegate Interface
type FilterHandler struct {
	Gate Interface
}

func (fh *FilterHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == http.MethodGet {
		response.Header().Set("Content-Type", "application/json")
		JSON, _ := json.Marshal(fh.Gate)
		fmt.Fprintf(response, `%s`, JSON)
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
		var message FilterRequest
		msgBytes, err := ioutil.ReadAll(request.Body)
		request.Body.Close()

		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not read request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if err := json.Unmarshal(msgBytes, &message); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not unmarshal request body", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if allow, err := checkRequestDetails(method, message, fh.Gate); !allow {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), err)
			xhttp.WriteError(response, http.StatusBadRequest, err)
			return
		}

		if method == http.MethodPost || method == http.MethodPut {
			if _, created := fh.Gate.SetFilter(message.Key, message.Values); created {
				response.WriteHeader(http.StatusCreated)
			} else {
				response.WriteHeader(http.StatusOK)
			}
		} else if method == http.MethodDelete {
			fh.Gate.DeleteFilter(message.Key)
			response.WriteHeader(http.StatusOK)
		}

		filtersJSON, _ := json.Marshal(fh.Gate)
		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", string(filtersJSON))
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(response, `%s`, filtersJSON)
	}
}

func checkRequestDetails(methodType string, f FilterRequest, gate Interface) (bool, error) {
	if len(f.Key) == 0 {
		return false, errors.New("missing filter key")
	}

	if methodType == http.MethodPost || methodType == http.MethodPut {
		if len(f.Values) == 0 {
			return false, errors.New("missing filter values")
		}

		if allowedFilters, allowedFiltersFound := gate.GetAllowedFilters(); allowedFiltersFound {
			if !allowedFilters.Has(f.Key) {
				allowedFiltersJSON, _ := json.Marshal(allowedFilters)
				return false, fmt.Errorf("filter key %s is not allowed. Allowed filters: %s", f.Key, allowedFiltersJSON)
			}
		}
	}

	return true, nil
}

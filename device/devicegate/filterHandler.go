package devicegate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
)

const gateKey = "gate"

// FilterHandler is an http.Handler that can get, add, and delete filters from a devicegate Interface
type FilterHandler struct {
	Gate Interface
}

type GateLogger struct {
	Logger log.Logger
}

func (fh *FilterHandler) GetFilters(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	JSON, _ := json.Marshal(fh.Gate)
	fmt.Fprintf(response, `%s`, JSON)
}

func (fh *FilterHandler) UpdateFilters(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())

	message, err := validateRequestBody(request)

	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error with request body", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if allow, err := checkRequestDetails(message, fh.Gate, true); !allow {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if _, created := fh.Gate.SetFilter(message.Key, message.Values); created {
		response.WriteHeader(http.StatusCreated)
	} else {
		response.WriteHeader(http.StatusOK)
	}

	newCtx := context.WithValue(request.Context(), gateKey, fh.Gate)
	request = request.Clone(newCtx)
	fmt.Printf("Checking key: %s", request.Context().Value(gateKey))
}

func (fh *FilterHandler) DeleteFilter(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())

	message, err := validateRequestBody(request)

	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error with request body", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if allow, err := checkRequestDetails(message, fh.Gate, false); !allow {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	fh.Gate.DeleteFilter(message.Key)
	response.WriteHeader(http.StatusOK)

	newCtx := context.WithValue(request.Context(), gateKey, fh.Gate)
	request = request.Clone(newCtx)
}

func (gl GateLogger) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(response, request)

		gl.Logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "context", "context details", request.Context())
		if gate, ok := request.Context().Value(gateKey).(Interface); ok {
			if filtersJSON, err := json.Marshal(gate); err == nil {
				response.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(response, `%s`, filtersJSON)
				gl.Logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", string(filtersJSON))
			} else {
				gl.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error with unmarshalling gate", logging.ErrorKey(), err)
			}
		} else {
			gl.Logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate not found in request context")
		}

	})

}

func validateRequestBody(request *http.Request) (FilterRequest, error) {
	var message FilterRequest
	msgBytes, err := ioutil.ReadAll(request.Body)
	request.Body.Close()

	if err != nil {
		return message, err
	}

	if e := json.Unmarshal(msgBytes, &message); e != nil {
		return message, e
	}

	return message, nil

}

func checkRequestDetails(f FilterRequest, gate Interface, checkFilterValues bool) (bool, error) {
	if len(f.Key) == 0 {
		return false, errors.New("missing filter key")
	}

	if checkFilterValues {
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

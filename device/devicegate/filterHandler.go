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

// FilterHandler is an http.Handler that can get, add, and delete filters from a devicegate Interface
type FilterHandler struct {
	Gate Interface
}

func (fh *FilterHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == http.MethodGet {
		response.Header().Set("Content-Type", "application/json")
		if JSON, err := json.Marshal(fh.Gate); err == nil {
			fmt.Fprintf(response, `%s`, JSON)
		} else {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not marshal filters into JSON", logging.ErrorKey(), err)
			xhttp.WriteError(response, http.StatusInternalServerError, err)
		}
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
		var message FilterRequest
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

		if method == http.MethodPost || method == http.MethodPut {

			if len(message.Values) == 0 {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no filter values found")
				xhttp.WriteErrorf(response, http.StatusBadRequest, "missing filter values")
				return
			}

			if allowedFilters, allowedFiltersFound := fh.Gate.GetAllowedFilters(); allowedFiltersFound {
				if !allowedFilters.Has(message.Key) {
					logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
					if allowedFiltersJSON, err := json.Marshal(allowedFilters); err == nil {
						xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %s", message.Key, allowedFiltersJSON)
					} else {
						xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed.", message.Key)
					}
					return

				}
			}

			if _, created := fh.Gate.SetFilter(message.Key, message.Values); created {
				response.WriteHeader(http.StatusCreated)
			} else {
				response.WriteHeader(http.StatusOK)
			}

		} else if method == http.MethodDelete {
			fh.Gate.DeleteFilter(message.Key)
			response.WriteHeader(http.StatusOK)
		}

		if filtersJSON, err := json.Marshal(fh.Gate); err == nil {
			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", string(filtersJSON))
			response.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(response, `%s`, filtersJSON)
		} else {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not marshal filters into JSON", logging.ErrorKey(), err)
		}
	}
}

// creates visitor function to convert filters to string
// func writeFilters(b *strings.Builder) func(string, Set) bool {
// 	var needsComma bool

// 	return func(key string, val Set) bool {
// 		if needsComma {
// 			b.WriteString(",\n")
// 			needsComma = false
// 		}

// 		fmt.Fprintf(b, `"%s": `, key)
// 		fmt.Fprintf(b, "%s", val.String())
// 		needsComma = true

// 		return true
// 	}
// }

// wrapper to build JSON string representation of filters
// func filtersToString(g Interface) string {
// 	var b strings.Builder
// 	var filtersBuilder strings.Builder
// 	b.WriteString("{")
// 	g.VisitAll(writeFilters(&filtersBuilder))

// 	if filtersBuilder.Len() > 0 {
// 		filtersBuilder.WriteString("\n")
// 		b.WriteString(filtersBuilder.String())
// 		filtersBuilder.WriteString("\n")
// 	}
// 	b.WriteString("}")
// 	return b.String()
// }

package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
	"github.com/xmidt-org/wrp-go/v2"
)

// Filter is an http.Handler which controls the filters of a gate.
type Filter struct {
	// Gate is the gate to add filters to
	Gate Interface
}

type FilterRequest struct {
	Key    string
	Values []string
}

const (
	partnerIDKey = "partner_id"
)

var (
	emptyVal  = struct{}{}
	filterMap = map[string]struct{}{
		partnerIDKey: emptyVal,
	}
)

func (f *Filter) ServeHTTP(response http.ResponseWriter, request *http.Request) {

	logger := logging.GetLogger(request.Context())

	method := request.Method
	if method == "GET" {
		filters := f.Gate.Filters()
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(response, `{"filters": %v}`, filters)
	} else if method == "POST" || method == "PUT" {
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
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "could not decode request body", logging.ErrorKey(), err)
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

		if filterMap != nil {
			_, ok := filterMap[message.Key]

			if !ok {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
				xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %v", message.Key, filterMap)
				return
			}
		}

		f.Gate.EditFilters(message.Key, message.Values, true)

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", f.Gate.Filters())

		response.WriteHeader(http.StatusOK)
	}
}

func RequestToWRP(req *http.Request) (*wrp.Message, error) {

	var message wrp.Message
	if req.Body != nil {
		msgBytes, err := ioutil.ReadAll(req.Body)
		req.Body.Close()

		// Write in what you just read
		req.Body = ioutil.NopCloser(bytes.NewBuffer(msgBytes))

		if err != nil {
			return nil, err
		} else {
			e := wrp.NewDecoderBytes(msgBytes, wrp.Msgpack).Decode(&message)
			if e != nil {
				return nil, e
			}

			return &message, nil
		}
	}

	return nil, nil
}

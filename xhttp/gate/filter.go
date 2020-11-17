package gate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

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

type filterRequest struct {
	Key    string
	Values []string
}

type FiltersStore struct {
	filters map[string]map[string]struct{}
	lock    sync.RWMutex
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

		if filterMap != nil {
			_, ok := filterMap[message.Key]

			if !ok {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "filter key is not allowed", "key: ", message.Key)
				xhttp.WriteErrorf(response, http.StatusBadRequest, "filter key %s is not allowed. Allowed filters: %v", message.Key, filterMap)
				return
			}
		}

		f.Gate.Filters().EditFilters(message.Key, message.Values, true)

		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate filters updated", "filters", f.Gate.Filters())

		response.WriteHeader(http.StatusOK)
	}
}

func RequestToWRP(req *http.Request) (*wrp.Message, error) {

	var message wrp.Message
	if req.Body != nil {
		msgBytes, err := ioutil.ReadAll(req.Body)
		req.Body.Close()

		// Write in what was just read
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

func (f *FiltersStore) Prettify() map[string][]string {
	copy := make(map[string][]string)

	f.lock.RLock()
	for k, v := range f.filters {
		var filters []string
		for filter := range v {
			filters = append(filters, filter)
		}

		copy[k] = filters
	}
	f.lock.RUnlock()

	return copy
}

func (f *FiltersStore) Filters() map[string]map[string]struct{} {
	filtersCopy := make(map[string]map[string]struct{})

	f.lock.RLock()
	for key, val := range f.filters {
		valCopy := make(map[string]struct{})

		for k, v := range val {
			valCopy[k] = v
		}

		filtersCopy[key] = valCopy
	}
	f.lock.RUnlock()

	return filtersCopy
}

//allows for adding and removing filters
//Will completely overwrite current filters in place
func (f *FiltersStore) EditFilters(key string, values []string, add bool) {
	f.lock.Lock()
	if add {
		valuesMap := make(map[string]struct{})

		for _, v := range values {
			valuesMap[v] = emptyVal
		}

		f.filters[key] = valuesMap
	} else {

		_, ok := f.filters[key]

		if ok {
			delete(f.filters, key)
		}
	}
	f.lock.Unlock()
}

func (f *FiltersStore) FilterRequest(m *Metadata) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	for filterKey, filterValues := range f.filters {
		switch filterKey {
		case partnerIDKey:
			if filterMatch(filterValues, msg.PartnerIDs...) {
				return true
			}
		}
	}

	return true
}

func filterMatch(filterValues map[string]struct{}, paramsToCheck ...string) bool {
	for _, param := range paramsToCheck {
		_, found := filterValues[param]

		if found {
			return true
		}
	}

	return false

}

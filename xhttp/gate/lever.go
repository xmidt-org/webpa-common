package gate

import (
	"net/http"
	"strconv"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/go-kit/kit/log/level"
)

// Lever is an http.Handler which controls the state of a gate.
type Lever struct {
	// Gate is the gate this lever controls
	Gate Interface

	// Parameter is the HTTP parameter, which must be a bool, used to set the state of the gate
	Parameter string
}

func (l *Lever) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := logging.GetLogger(request.Context())

	if err := request.ParseForm(); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "bad form request", logging.ErrorKey(), err)
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	v := request.FormValue(l.Parameter)
	if len(v) == 0 {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no parameter found", "parameter", l.Parameter)
		xhttp.WriteErrorf(response, http.StatusBadRequest, "Missing %s parameter", l.Parameter)
		return
	}

	f, err := strconv.ParseBool(v)
	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "parameter is not a bool", "parameter", l.Parameter, logging.ErrorKey(), err)
		xhttp.WriteErrorf(response, http.StatusBadRequest, "The %s parameter must be a bool", l.Parameter)
		return
	}

	var changed bool
	if f {
		changed = l.Gate.Raise()
	} else {
		changed = l.Gate.Lower()
	}

	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "gate update", "open", f, "changed", changed)

	if changed {
		response.WriteHeader(http.StatusCreated)
	} else {
		response.WriteHeader(http.StatusOK)
	}
}

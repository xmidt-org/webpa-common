// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"net/http"
	"strconv"

	"github.com/xmidt-org/sallust/sallusthttp"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"go.uber.org/zap"
)

// Lever is an http.Handler which controls the state of a gate.
type Lever struct {
	// Gate is the gate this lever controls
	Gate Interface

	// Parameter is the HTTP parameter, which must be a bool, used to set the state of the gate
	Parameter string
}

func (l *Lever) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	logger := sallusthttp.Get(request)

	if err := request.ParseForm(); err != nil {
		logger.Error("bad form request", zap.Error(err))
		xhttp.WriteError(response, http.StatusBadRequest, err)
		return
	}

	v := request.FormValue(l.Parameter)
	if len(v) == 0 {
		logger.Error("no parameter found", zap.String("parameter", l.Parameter))
		xhttp.WriteErrorf(response, http.StatusBadRequest, "missing %s parameter", l.Parameter)
		return
	}

	f, err := strconv.ParseBool(v)
	if err != nil {
		logger.Error("parameter is not a bool", zap.String("parameter", l.Parameter), zap.Error(err))
		xhttp.WriteErrorf(response, http.StatusBadRequest, "the %s parameter must be a bool", l.Parameter)
		return
	}

	var changed bool
	if f {
		changed = l.Gate.Raise()
	} else {
		changed = l.Gate.Lower()
	}

	logger.Info("gate update", zap.Bool("open", f), zap.Bool("changed", changed))

	if changed {
		response.WriteHeader(http.StatusCreated)
	} else {
		response.WriteHeader(http.StatusOK)
	}
}

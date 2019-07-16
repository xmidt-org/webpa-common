package servicehttp

import (
	"context"
	"fmt"
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/xhttp"
)

// KeyFromHeader produces a go-kit decoder which expects an HTTP header to contain the service key.
func KeyFromHeader(header string, parser service.KeyParser) gokithttp.DecodeRequestFunc {
	if len(header) == 0 {
		panic("A header is required")
	}

	if parser == nil {
		panic("A parser is required")
	}

	missingHeader := &xhttp.Error{
		Code: http.StatusBadRequest,
		Text: fmt.Sprintf("missing %s header", header),
	}

	return func(_ context.Context, r *http.Request) (interface{}, error) {
		v := r.Header.Get(header)
		if len(v) == 0 {
			return nil, missingHeader
		}

		return parser(v)
	}
}

// KeyFromPath uses a gorilla/mux path variable as the source for the service Key.
func KeyFromPath(variable string, parser service.KeyParser) gokithttp.DecodeRequestFunc {
	if len(variable) == 0 {
		panic("A variable is required")
	}

	if parser == nil {
		panic("A parser is required")
	}

	noPathVariables := &xhttp.Error{
		Code: http.StatusInternalServerError,
		Text: "no path variables found",
	}

	missingValue := &xhttp.Error{
		Code: http.StatusBadRequest,
		Text: fmt.Sprintf("missing path variable %s", variable),
	}

	return func(_ context.Context, r *http.Request) (interface{}, error) {
		vars := mux.Vars(r)
		if len(vars) == 0 {
			return nil, noPathVariables
		}

		v, ok := vars[variable]
		if !ok {
			return nil, missingValue
		}

		return parser(v)
	}
}

package fanouthttp

import (
	"context"
	"net/http"
	"net/textproto"

	"github.com/Comcast/webpa-common/middleware/fanout"
	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// VariablesToHeaders returns a go-kit RequestFunc that emits path variables from the original fanout request
// into headers of a component request.  The *fanoutRequest must be present in the context.
//
// The number of variadic arguments must be even, or this function panics.  Each pair of variadic arguments
// maps a gorilla/mux path variable to a corresponding HTTP header, e.g. Variables("id", "X-Id").
func VariablesToHeaders(first, second string, rest ...string) gokithttp.RequestFunc {
	if len(rest)%2 != 0 {
		panic("The number of extra values to this function must be even")
	}

	mapping := make(map[string]string, 1+len(rest)/2)
	mapping[first] = textproto.CanonicalMIMEHeaderKey(second)
	for i := 0; i < len(rest); i += 2 {
		mapping[rest[i]] = textproto.CanonicalMIMEHeaderKey(rest[i+1])
	}

	return func(ctx context.Context, r *http.Request) context.Context {
		if fr, ok := fanout.FromContext(ctx).(*fanoutRequest); ok {
			pathVars := mux.Vars(fr.original)
			if len(pathVars) > 0 {
				for variable, header := range mapping {
					if value, ok := pathVars[variable]; ok {
						r.Header[header] = []string{value}
					}
				}
			}
		}

		return ctx
	}
}

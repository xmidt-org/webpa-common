package servicehttp

import (
	"context"
	"net/http"
	"strings"

	money "github.com/Comcast/golang-money"
	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log/level"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// Redirect returns a go-kit EncodeResponseFunc that redirects to the instance hashed by the accessor.
// If the original requestURI is populated under the go-kit key ContextKeyRequestURI, it is appended to
// the hashed instance.
func Redirect(redirectCode int) gokithttp.EncodeResponseFunc {
	if redirectCode < 300 {
		redirectCode = http.StatusTemporaryRedirect
	}

	return func(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
		var (
			logger        = logging.GetLogger(ctx)
			instance      = response.(string)
			requestURI, _ = ctx.Value(gokithttp.ContextKeyRequestURI).(string)
		)

		if len(requestURI) > 0 {
			instance = instance + strings.TrimRight(requestURI, "/")
		}

		httpTracker, ok := money.TrackerFromContext(ctx)
		if ok {
			result, _ := httpTracker.Finish()
			money.WriteMoneySpanHeaders(result, rw)
		}

		logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "redirecting", "instance", instance)
		rw.Header().Set("Location", instance)
		rw.WriteHeader(redirectCode)

		return nil
	}
}

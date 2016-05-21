package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"golang.org/x/net/context"
	"net/http"
)

// Recover is a ChainHandler that recovers from panics down the chain.  If a panic occurs,
// a well-formed HTTP response is written out.
//
// This ChainHandler should normally be at the start of the chain.
func Recover(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fact.MustLogger(ctx).Error("Recovered: %v", recovered)
			WriteError(
				response,
				fmt.Sprintf("%v", recovered),
			)
		}
	}()

	next.ServeHTTP(ctx, response, request)
}

package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/fact"
	"net/http"
)

func Convey() ChainHandler {
	return ConveyCustom(ConveyHeader, base64.StdEncoding)
}

func ConveyCustom(conveyHeader string, encoding *base64.Encoding) ChainHandler {
	return ChainHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
		logger, hasLogger := fact.Logger(ctx)
		rawPayload := request.Header.Get(conveyHeader)
		if len(rawPayload) > 0 {
			// BUG: https://www.teamccp.com/jira/browse/WEBPA-787
			const notAvailable string = "not-available"
			if rawPayload == notAvailable {
				if hasLogger {
					logger.Error("Invalid convey header: %s.  FIX ME: https://www.teamccp.com/jira/browse/WEBPA-787", rawPayload)
				}
			} else if conveyPayload, err := device.ParseConvey(rawPayload, encoding); err != nil {
				message := fmt.Sprintf(InvalidConveyPattern, rawPayload, err)
				if hasLogger {
					logger.Error(message)
				}

				WriteJsonError(
					response,
					http.StatusBadRequest,
					message,
				)

				return
			} else {
				ctx = fact.SetConvey(ctx, conveyPayload)
			}
		}

		next.ServeHTTP(ctx, response, request)
	})
}

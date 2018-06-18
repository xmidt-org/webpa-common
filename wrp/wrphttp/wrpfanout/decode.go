package wrpfanout

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrphttp"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/fanout"
)

func DecodeHeaders(output wrp.Format) fanout.RequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, originalBody []byte) (context.Context, error) {
		var message wrp.Message
		if err := wrphttp.SetMessageFromHeaders(original.Header, &message); err != nil {
			return ctx, err
		}

		if len(originalBody) > 0 {
			message.Payload = originalBody
			if len(message.ContentType) == 0 {
				message.ContentType = "application/octet-stream"
			}
		}

		var buffer []byte
		if err := wrp.NewEncoderBytes(&buffer, output).Encode(&message); err != nil {
			return ctx, err
		}

		fanout.Header.Set("Content-Type", output.ContentType())
		fanout.Body, fanout.GetBody = xhttp.NewRewindBytes(buffer)
		fanout.ContentLength = int64(len(buffer))
		return ctx, nil
	}
}

func DecodeBody(output wrp.Format) fanout.RequestFunc {
	return func(ctx context.Context, original, fanout *http.Request, originalBody []byte) (context.Context, error) {
		return ctx, nil
	}
}

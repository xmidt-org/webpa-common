package transporthttp

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
)

// GetBody is a go-kit RequestFunc that sets the request's GetBody function.  This ensures
// that redirects are properly followed automatically.
func GetBody(ctx context.Context, request *http.Request) context.Context {
	if body, err := ioutil.ReadAll(request.Body); err == nil {
		request.Body = ioutil.NopCloser(bytes.NewReader(body))
		request.GetBody = func() (send io.ReadCloser, err error) {
			return ioutil.NopCloser(bytes.NewReader(body)), nil
		}
	}

	return ctx
}

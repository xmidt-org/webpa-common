package client

import (
	"net/http"

	"github.com/Comcast/webpa-common/xhttp"
)

type WebPAClient struct {
	measures OutboundMeasures
	client   func(*http.Request) (*http.Response, error)
}

func NewWebPAClient(om OutboundMeasures, t func(*http.Request) (*http.Response, error)) *WebPAClient {
	return &WebPAClient{
		measures: om,
		client:   t,
	}
}

func (w *WebPAClient) WithTransactor(t func(*http.Request) (*http.Response, error)) {
	w.client = t
}

func (w *WebPAClient) Transact(r *http.Request) (*http.Response, error) {
	return w.client(r)
}

func (w *WebPAClient) RetryTransact(r *http.Request, ro xhttp.RetryOptions) (*http.Response, error) {
	return xhttp.RetryTransactor(ro, w.client)(r)
}

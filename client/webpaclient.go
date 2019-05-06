package client

import (
	"errors"
	"net/http"
	"sync"

	"github.com/Comcast/webpa-common/xhttp"
)

type WebPAClient struct {
	m        sync.RWMutex
	measures OutboundMeasures
	client   func(*http.Request) (*http.Response, error)
}

func NewWebPAClient(om OutboundMeasures, t func(*http.Request) (*http.Response, error)) *WebPAClient {
	return &WebPAClient{
		measures: om,
		client:   t,
	}
}

func (w *WebPAClient) ChangeTransactor(t func(*http.Request) (*http.Response, error)) {
	defer w.m.Unlock()
	w.m.Lock()

	w.client = t
	return
}

func (w *WebPAClient) Transact(r *http.Request) (*http.Response, error) {
	return w.client(r)
}

func (w *WebPAClient) TransactWith(t func(*http.Request) (*http.Response, error), r *http.Request) (*http.Response, error) {
	return t(r)
}

func (w *WebPAClient) RetryTransact(r *http.Request, ro *xhttp.RetryOptions) (*http.Response, error) {
	if w != nil && ro == nil {
		return nil, errors.New("Need non empty retry options")
	}

	res, err := xhttp.RetryTransactor(*ro, w.client)(r)
	if err != nil {
		return nil, err
	}

	return res, nil
}

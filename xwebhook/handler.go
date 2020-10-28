package xwebhook

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
)

func newAddWebhookHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		decodeAddWebhookRequest,
		encodeAddWebhookResponse,
	)
}

func newGetAllWebhooksHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		decodeGetAllWebhooksRequest,
		encodeGetAllWebhooksResponse,
	)
}

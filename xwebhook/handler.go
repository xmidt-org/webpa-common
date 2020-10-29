package xwebhook

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
)

func NewAddWebhookHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		decodeAddWebhookRequest,
		encodeAddWebhookResponse,
	)
}

func NewGetAllWebhooksHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		decodeGetAllWebhooksRequest,
		encodeGetAllWebhooksResponse,
	)
}

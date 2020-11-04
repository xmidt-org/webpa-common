package xwebhook

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func newAddWebhookEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(*addWebhookRequest)

		return nil, s.Add(r.owner, r.webhook)
	}
}

func newGetAllWebhooksEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(*getAllWebhooksRequest)
		return s.AllWebhooks(r.owner)
	}
}

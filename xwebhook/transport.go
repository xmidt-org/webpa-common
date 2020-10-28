package xwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xmidt-org/bascule"
)

const DEFAULT_EXPIRATION_DURATION time.Duration = time.Minute * 5

type getAllWebhooksRequest struct {
	owner string
}

func decodeGetAllWebhooksRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var owner string
	if auth, ok := bascule.FromContext(req.Context()); ok {
		owner = auth.Token.Principal()
	}
	return &getAllWebhooksRequest{
		owner: owner,
	}, nil
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	webhooks := response.([]Webhook)
	encoded_webhooks, err := json.Marshal(&webhooks)
	if err != nil {
		return err
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(encoded_webhooks)
}

func decodeAddWebhookRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	requestPayload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	webhook := new(Webhook)

	err = json.Unmarshal(requestPayload, webhook);
	if err != nil {
		webhook, err = getFirstFromList(requestPayload);
		if err != nil {
			return nil, err
		}
	}

	err = validateWebhook(webhook, r.RemoteAddr)
	if err != nil {
		nil, err
	}

	return webhook, nil
}

func encodeAddWebhookRequest(ctx context.Context, rw http.ResponseWriter, response interface {}) error {
	//TODO: 
	return nil
}

func getFirstFromList(requestPayload []byte) (*Webhook, error) {
	var webhooks []Webhook

	err := json.Unmarshal(requestPayload, &webhooks)
	if err != nil {
		return nil, err
	}

	if len(webhooks) < 1 {
		return nil, errors.New("No webhooks in request data list")
	}
	return &webhooks[0], nil
}

func validateWebhook(webhook *Webhook, requestAddress string) (err error) {
	if strings.TrimSpace(webhook.Config.URL) == "" {
		return errors.New("invalid Config URL")
	}

	if len(w.Events) == 0 {
		return errors.New("invalid events")
	}

	// TODO Validate content type ?  What about different types?

	if len(webhook.Matcher.DeviceId) == 0 {
		w.Matcher.DeviceId = []string{".*"} // match anything
	}

	if "" == webhook.Address && "" != requestAddress {
		// Record the IP address the request came from
		host, _, _err := net.SplitHostPort(requestAddress)
		if nil != _err {
			err = _err
			return
		}
		webhook.Address = host
	}

	// always set duration to default
	webhook.Duration = DEFAULT_EXPIRATION_DURATION

	if &webhook.Until == nil || webhook.Until.Equal(time.Time{}) {
		webhook.Until = time.Now().Add(webhook.Duration)
	}

	return
}

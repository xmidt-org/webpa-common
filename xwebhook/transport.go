package xwebhook

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"

	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/xhttp"
)

const defaultWebhookExpiration time.Duration = time.Minute * 5

const (
	contentTypeHeader string = "Content-Type"
	jsonContentType   string = "application/json"
)

type getAllWebhooksRequest struct {
	owner string
}

type addWebhookRequest struct {
	owner   string
	webhook *Webhook
}

func decodeGetAllWebhooksRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return &getAllWebhooksRequest{
		owner: getOwner(r),
	}, nil
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	webhooks := response.([]Webhook)
	obfuscateSecrets(webhooks)
	encodedWebhooks, err := json.Marshal(&webhooks)
	if err != nil {
		return err
	}

	rw.Header().Set(contentTypeHeader, jsonContentType)
	_, err = rw.Write(encodedWebhooks)
	return err
}

func decodeAddWebhookRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	requestPayload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	webhook := new(Webhook)

	err = json.Unmarshal(requestPayload, webhook)
	if err != nil {
		//TODO: we should get rid of this if we can. It's not listed in our swagger page but I'm keeping it just to
		// match the current behavior.
		webhook, err = getFirstFromList(requestPayload)
		if err != nil {
			return nil, err
		}
	}

	err = validateWebhook(webhook, r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	return &addWebhookRequest{
		owner:   getOwner(r),
		webhook: webhook,
	}, nil
}

func encodeAddWebhookResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	rw.Header().Set(contentTypeHeader, jsonContentType)
	rw.Write([]byte(`{"message": "Success"}`))
	return nil
}

func getOwner(r *http.Request) (owner string) {
	if auth, ok := bascule.FromContext(r.Context()); ok {
		owner = auth.Token.Principal()
	}
	return
}

func getFirstFromList(requestPayload []byte) (*Webhook, error) {
	var webhooks []Webhook

	err := json.Unmarshal(requestPayload, &webhooks)
	if err != nil {
		return nil, err
	}

	if len(webhooks) < 1 {
		return nil, &xhttp.Error{Text: "no webhooks in request data list", Code: http.StatusBadRequest}
	}
	return &webhooks[0], nil
}

func obfuscateSecrets(webhooks []Webhook) {
	for i := range webhooks {
		webhooks[i].Config.Secret = "<obfuscated>"
	}
}

func validateWebhook(webhook *Webhook, requestOriginAddress string) (err error) {
	if strings.TrimSpace(webhook.Config.URL) == "" {
		return &xhttp.Error{Code: http.StatusBadRequest, Text: "invalid Config URL"}
	}

	if len(webhook.Events) == 0 {
		return &xhttp.Error{Code: http.StatusBadRequest, Text: "invalid events"}
	}

	// TODO Validate content type ?  What about different types?

	if len(webhook.Matcher.DeviceID) == 0 {
		webhook.Matcher.DeviceID = []string{".*"} // match anything
	}

	if webhook.Address == "" && requestOriginAddress != "" {
		host, _, err := net.SplitHostPort(requestOriginAddress)
		if err != nil {
			return err
		}
		webhook.Address = host
	}

	// always set duration to default
	webhook.Duration = defaultWebhookExpiration

	if &webhook.Until == nil || webhook.Until.Equal(time.Time{}) {
		webhook.Until = time.Now().Add(webhook.Duration)
	}

	return nil
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set(contentTypeHeader, jsonContentType)
	code := http.StatusInternalServerError
	if sc, ok := err.(kithttp.StatusCoder); ok {
		code = sc.StatusCode()
	}
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message": err.Error(),
		})
}

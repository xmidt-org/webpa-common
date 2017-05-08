package webhooklisteners

import (
	"time"
)

// WebHookListener is the structure that represents the Webhook listener
// data we share.
type WebHookListener struct {

	// The URL to deliver messages to.
	URL string `json:"url"`

	// The URL to notify when we cut off a client due to overflow.
	// Optional, set to "" to disable behavior
	FailureURL string `json:"failure_url"`

	// The content-type to set the messages to (unless specified by WRP).
	ContentType string `json:"content_type"`

	// The secret to use for the SHA1 HMAC.
	// Optional, set to "" to disable behavior.
	Secret string `json:"secret,omitempty"`

	// The list of regular expressions to match event type against.
	Events []string `json:"events"`

	// The list of regular expressions to match against the metadata.
	Matchers map[string][]string `json:"matchers,omitempty"`

	// The specified duration for this hook to live
	Duration time.Duration `json:"duration"`

	// The absolute time when this hook is to be disabled
	Until time.Time `json:"until"`

	// The address that performed the registration
	Address string `json:"registered_from_address"`
}

// ID creates the canonical string identifing a WebhookListener
func (w *WebHookListener) ID() string {
	return w.URL
}

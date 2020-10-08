package xwebhook

import (
	"time"
)

// Subscription contains all the information needed to serve events to webhook subscribers.
type Subscription struct {
	// Address is the subscription request origin HTTP Address
	Address string `json:"registered_from_address"`

	// Config contains data to inform how events are delivered.
	Config struct {
		// URL is the HTTP URL to deliver messages to.
		URL string `json:"url"`

		// ContentType is content type value to set WRP messages to (unless already specified in the WRP).
		ContentType string `json:"content_type"`

		// Secret is the string value for the SHA1 HMAC.
		// (Optional, set to "" to disable behavior).
		Secret string `json:"secret,omitempty"`

		// AlternativeURLs is a list of explicit URLs that should be round robin through on failure cases to the main URL.
		AlternativeURLs []string `json:"alt_urls,omitempty"`
	} `json:"config"`

	// FailureURL is the URL used to notify subscribers when they've been cut off due to event overflow.
	// Optional, set to "" to disable notifications.
	FailureURL string `json:"failure_url"`

	// Events is the list of regular expressions to match an event type against.
	Events []string `json:"events"`

	// Matcher type contains values to match against the metadata.
	Matcher struct {
		// DeviceID is the list of regular expressions to match device id type against.
		DeviceID []string `json:"device_id"`
	} `json:"matcher,omitempty"`

	// Duration describes how long the subscription lasts once added.
	// Deprecated. User input is ignored and value is always 5m.
	Duration time.Duration `json:"duration"`

	// Until describes the time this subscription expires.
	Until time.Time `json:"until"`
}

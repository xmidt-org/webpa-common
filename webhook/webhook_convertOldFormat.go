package webhook

import (
	"encoding/json"
	"time"
)

type oldW struct {
	// Configuration for message delivery
	Config struct {
		// The URL to deliver messages to.
		URL string `json:"url"`

		// The content-type to set the messages to (unless specified by WRP).
		ContentType string `json:"content_type"`

		// The secret to use for the SHA1 HMAC.
		// Optional, set to "" to disable behavior.
		Secret string `json:"secret,omitempty"`
	} `json:"config"`

	// The list of regular expressions to match event type against.
	Events []string `json:"events"`

	// Matcher type contains values to match against the metadata.
	Matcher struct {
		// The list of regular expressions to match device id type against.
		DeviceId []string `json:"device_id"`
	} `json:"matcher,omitempty"`

	// The specified duration for this hook to live
	Duration int64 `json:"duration"`

	// The absolute time when this hook is to be disabled
	Until int64 `json:"until"`

	// The address that performed the registration
	Address string `json:"registered_from_address"`
}

func convertOldHooksToNewHooks(body []byte) (hooks []W, err error) {
	var oldHooks []oldW
	err = json.Unmarshal(body, &oldHooks)
	if err != nil {
		return
	}

	for _, oldHook := range oldHooks {
		var tempHook W
		tempHook.Config.URL = oldHook.Config.URL
		tempHook.Config.ContentType = oldHook.Config.ContentType
		tempHook.Config.Secret = oldHook.Config.Secret
		tempHook.Events = oldHook.Events
		tempHook.Matcher = oldHook.Matcher
		tempHook.Duration = time.Duration(oldHook.Duration) * time.Second
		tempHook.Until = time.Unix(oldHook.Until, 0)
		tempHook.Address = oldHook.Address
		
		hooks = append(hooks, tempHook)
	}
	
	return
}
